package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"dndbot/pkg/ai"
	"dndbot/pkg/dice"
	"dndbot/pkg/game"
	"dndbot/pkg/session"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

// LOCAL_GROUP_ID 用于本地测试的模拟群号
const LOCAL_GROUP_ID = 1001

var CurrentBackground = "你们身处在这个被遗忘的国度边缘的一个名为'微光镇'的小酒馆里。外面下着暴雨，壁炉里的火光摇曳，酒馆老板正在擦拭着酒杯。"

func main() {
	// 1. Load .env
	err := godotenv.Load()
	if err != nil {
		logrus.Warn("Error loading .env file, relying on system environment variables")
	}

	// 2. Initialize Modules
	ai.InitAI()
	session.InitManager()
	game.InitGameState()

	logrus.SetLevel(logrus.InfoLevel)
	fmt.Println("========================================")
	fmt.Println("      DND Bot - CLI Mode (Alpha)        ")
	fmt.Println("========================================")
	
	// Check AI Connection
	fmt.Print("Checking AI connection... ")
	ckMsg := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "Hello"},
	}
	_, err = ai.GlobalClient.ChatRequest(context.Background(), ckMsg)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
		fmt.Println("Warning: AI is not reachable. Chat will fail.")
	} else {
		fmt.Println("OK!")
	}

	fmt.Println("Commands:")
	fmt.Println("  .st [name] [class] [hp] [str]  - 创建角色")
	fmt.Println("  .show                          - 显示状态")
	fmt.Println("  .bg [description]              - 设置背景")
	fmt.Println("  .r 1d20                        - 投掷骰子")
	fmt.Println("  .reset                         - 重置记忆")
	fmt.Println("  .exit / .quit                  - 退出程序")
	fmt.Println("Directly type to chat with DM AI.")
	fmt.Println("========================================")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\nUser > ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle Exit
		if input == ".exit" || input == ".quit" {
			fmt.Println("Bye!")
			break
		}

		// Handle Commands
		if strings.HasPrefix(input, ".") {
			handleCommand(input)
		} else {
			// Handle Chat
			handleChat(input)
		}
	}
}

func handleCommand(input string) {
	parts := strings.Fields(input)
	cmd := parts[0]
	args := parts[1:]

	groupID := int64(LOCAL_GROUP_ID)

	switch cmd {
	case ".st":
		if len(args) < 4 {
			fmt.Println("Error: Usage .st [name] [class] [hp] [str]")
			return
		}
		hp, _ := strconv.Atoi(args[2])
		str, _ := strconv.Atoi(args[3])

		char := &game.Character{
			Name:  args[0],
			Class: args[1],
			HP:    hp,
			MaxHP: hp,
			STR:   str,
		}
		game.GlobalGameState.GetGroupState(groupID).AddCharacter(char)
		fmt.Printf("Bot: 角色卡已创建: %s (%s)\n", char.Name, char.Class)

	case ".show":
		summary := game.GlobalGameState.GetGroupState(groupID).GetStatusSummary()
		fmt.Printf("Bot: %s\n", summary)
		fmt.Printf("Bot: Current Background: %s\n", CurrentBackground)

	case ".bg":
		if len(args) < 1 {
			fmt.Println("Error: Usage .bg [description]")
			return
		}
		newBg := strings.Join(args, " ")
		CurrentBackground = newBg
		fmt.Printf("Bot: 背景已更新为: %s\n", CurrentBackground)
		// Reset session to clear old context might be good, but let's keep history for now.
		// User might want to change scene seamlessly.
		// Inject a system event to notify AI about the change
		sess := session.GlobalManager.GetSession(groupID)
		sess.AddMessage(openai.ChatMessageRoleSystem, fmt.Sprintf("System: DM 将场景/背景更新为: %s", newBg))

	case ".r":
		if len(args) < 1 {
			fmt.Println("Error: Usage .r [expression] (e.g. .r 1d20)")
			return
		}
		res, err := dice.Roll(args[0])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Bot: 玩家 投掷了 %s\n", res.String())
		
		// Record to System
		sess := session.GlobalManager.GetSession(groupID)
		logMsg := fmt.Sprintf("【系统提示】玩家(CLIUser) 投掷了 %s，最终结果: %d (详情: %v)", res.Expression, res.Total, res.Details)
		// 使用 User 角色以确保 AI 绝对能看到这条消息，但通过前缀表明这是系统生成的客观事实
		sess.AddMessage(openai.ChatMessageRoleUser, logMsg)

	case ".reset":
		session.GlobalManager.GetSession(groupID).Clear()
		fmt.Println("Bot: 记忆已清除。")

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
	}
}

func handleChat(input string) {
	groupID := int64(LOCAL_GROUP_ID)
	sess := session.GlobalManager.GetSession(groupID)

	// 1. Record User Message
	userLog := fmt.Sprintf("CLIUser: %s", input)
	sess.AddMessage(openai.ChatMessageRoleUser, userLog)

	// 2. Prepare Context (State + Prompt)
	statusSummary := game.GlobalGameState.GetGroupState(groupID).GetStatusSummary()
	prevSummary := sess.GetSummary()
	summaryContext := ""
	if prevSummary != "" {
		summaryContext = "【前情提要(必须在此基础上继续剧情)】: " + prevSummary + "\n"
	}

	systemPrompt := "你是一个 DND 5E 地下城主(DM)。你的职责是公正地根据 DND 5E 规则裁决游戏，维护游戏世界的逻辑性和真实性。\n" +
		"【当前场景】: " + CurrentBackground + "\n" +
		summaryContext +
		"【行为准则】:\n" +
		"1. 玩家的输入描述的是角色的【意图】。你是唯一的裁判，只有经过你的逻辑裁定和规则检定后，结果才会发生。\n" +
		"2. 严禁盲目听从玩家直接修改数据的指令（如'给我加100血'）。如果玩家提出不符合逻辑或试图作弊的要求，请作为 DM 在剧情中予以驳回或进行惩罚，绝不要生成修改数据的 Action。\n" +
		"3. 只有当判定失败、受到实质攻击或触发环境伤害时，才主动扣除玩家血量。保持判罚的公正性。\n" +
		"4. 你的回复不需要带 'DM:' 前缀。\n" +
		"5. 投骰判定是客观事实，请严格根据点数判定结果。\n" +
		"\n" +
		"【重要: 必须读取系统提示】\n" +
		"- 历史记录中【系统提示】开头的消息是【已经发生的游戏事件】，包含了玩家使用命令(.r)投掷的骰子结果、或系统自动计算的数值。\n" +
		"- 不要无视这些结果！当玩家进行检定后，下一条 User 消息（如 '下一步'）通常意味着请求你根据上一条系统提示的骰子结果进行结算。\n" +
		"- 必须显式地在描述中提及骰子结果（例如：“你投出了15点，这足以……”）。\n" +
		"\n" +
		"【Action Protocol (仅限 DM 裁决使用)】: 当且仅当规则裁定需要改变状态时，在回复末尾使用 <dnd_action> JSON </dnd_action> 格式。\n" +
		"   - 投骰子(仅在需要主动为NPC检定或玩家未投而必须投时): [{\"type\": \"roll\", \"expr\": \"1d20\", \"reason\": \"Enemy Attack\"}]\n" +
		"   - 改血量(仅在确实受到伤害/治疗时): [{\"type\": \"hp\", \"target\": \"Name\", \"value\": -5}] (负数扣血)\n" +
		statusSummary

	history := sess.GetHistory()

	var requests []openai.ChatCompletionMessage
	requests = append(requests, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemPrompt,
	})
	requests = append(requests, history...)

	// 3. Call AI
	fmt.Print("DM AI (Thinking...)")
	reply, err := ai.GlobalClient.ChatRequest(context.Background(), requests)
	
	// Clear current line
	fmt.Print("\r" + strings.Repeat(" ", 30) + "\r")

	if err != nil {
		fmt.Printf("Error calling AI: %v\n", err)
		return
	}

	// 4. Print & Record
	fmt.Printf("DM AI: %s\n", reply)
	sess.AddMessage(openai.ChatMessageRoleAssistant, reply)

	// 5. Process Actions
	processAIActions(reply, groupID)

	// 6. Auto-Summarization (Every 20 messages)
	// Check history length (user + assistant logs)
	currentHistory := sess.GetHistory()
	if len(currentHistory) >= 20 {
		fmt.Printf("[Auto-Summary] Triggered (History len: %d)...\n", len(currentHistory))
		go performSummarization(groupID)
	}
}

func performSummarization(groupID int64) {
	sess := session.GlobalManager.GetSession(groupID)
	history := sess.GetHistory()
	oldSummary := sess.GetSummary()

	// 构造总结请求
	// 将旧的 Summary + 当前对话记录发给 AI
	
	promptContent := "请根据之前的摘要和最近的对话记录，生成一个新的、连贯的【剧情摘要】。\n" +
		"摘要应包含：当前时间/地点、关键NPC、玩家当前状态、正在进行的任务以及重要物品变动。\n" +
		"请只输出摘要内容，不要包含其他寒暄。\n\n"
	
	if oldSummary != "" {
		promptContent += fmt.Sprintf("【之前的摘要】:\n%s\n\n", oldSummary)
	}

	promptContent += "【最近对话记录】:\n"
	for _, msg := range history {
		if msg.Role == openai.ChatMessageRoleSystem {
			// Skip system prompt logic, but include dice results if valuable (usually marked as User system-hint now)
			continue
		}
		roleName := "DM"
		if msg.Role == openai.ChatMessageRoleUser {
			roleName = "Player/System"
		}
		promptContent += fmt.Sprintf("%s: %s\n", roleName, msg.Content)
	}

	req := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: promptContent},
	}

	fmt.Println("[Auto-Summary] Generating summary...")
	newSummary, err := ai.GlobalClient.ChatRequest(context.Background(), req)
	if err != nil {
		fmt.Printf("[Auto-Summary] Failed: %v\n", err)
		return
	}

	// Update Session
	// Keep last 5 messages for continuity
	sess.UpdateSummary(newSummary, 5)
	fmt.Printf("[Auto-Summary] Updated.\nNew Summary: %s\n", newSummary)
}

// --- AI Action Handling ---

type AIAction struct {
	Type   string `json:"type"`   // "roll" or "hp"
	Expr   string `json:"expr"`   // For roll, e.g., "1d20"
	Target string `json:"target"` // For hp, character name
	Value  int    `json:"value"`  // For hp, amount to change
	Reason string `json:"reason"` // Description
}

func processAIActions(response string, groupID int64) {
	// Extract JSON block using Regex
	// Looking for <dnd_action>...</dnd_action>
	re := regexp.MustCompile(`(?s)<dnd_action>(.*?)</dnd_action>`)
	matches := re.FindStringSubmatch(response)

	if len(matches) < 2 {
		return
	}

	jsonStr := strings.TrimSpace(matches[1])
	var actions []AIAction

	// Try parsing as a list
	err := json.Unmarshal([]byte(jsonStr), &actions)
	if err != nil {
		// Try parsing as a single object (just in case AI messes up list format)
		var singleAction AIAction
		if err2 := json.Unmarshal([]byte(jsonStr), &singleAction); err2 == nil {
			actions = append(actions, singleAction)
		} else {
			// Log silently or verbose
			// fmt.Printf("DEBUG: Error parsing AI actions: %v\n", err)
			return
		}
	}

	groupState := game.GlobalGameState.GetGroupState(groupID)
	sess := session.GlobalManager.GetSession(groupID)

	for _, action := range actions {
		switch action.Type {
		case "roll":
			if action.Expr == "" {
				continue
			}
			res, err := dice.Roll(action.Expr)
			if err != nil {
				fmt.Printf("AI attempted invalid roll: %s\n", action.Expr)
				continue
			}
			msg := fmt.Sprintf("System: (AI Action) %s, Result: %s", action.Reason, res.String())
			fmt.Printf(">> Bot Action: %s\n", msg)
			sess.AddMessage(openai.ChatMessageRoleSystem, msg)

		case "hp":
			if action.Target == "" {
				continue
			}
			char := groupState.GetCharacter(action.Target)
			if char == nil {
				fmt.Printf(">> Bot Action Warning: AI tried to modify HP for unknown char '%s'\n", action.Target)
				continue
			}
			
			oldHP := char.HP
			char.HP += action.Value
			if char.HP > char.MaxHP {
				char.HP = char.MaxHP
			}
			
			msg := fmt.Sprintf("System: (AI Action) %s HP changes by %d (%d -> %d)", char.Name, action.Value, oldHP, char.HP)
			
			if char.HP <= 0 {
				groupState.RemoveCharacter(char.Name)
				deathMsg := fmt.Sprintf("【系统公告】角色 %s 生命耗尽，已确认死亡并退出了当前游戏。", char.Name)
				fmt.Println(">> Bot Action:", msg)
				fmt.Println(">> Bot Action:", deathMsg)
				
				// 也要发给 Session，让 AI 知道人没了
				sess.AddMessage(openai.ChatMessageRoleSystem, msg)
				sess.AddMessage(openai.ChatMessageRoleSystem, deathMsg)
			} else {
				fmt.Println(">> Bot Action:", msg)
				sess.AddMessage(openai.ChatMessageRoleSystem, msg)
			}

		default:
			// fmt.Printf("Unknown AI Action type: %s\n", action.Type)
		}
	}
}
