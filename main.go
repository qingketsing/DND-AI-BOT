package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"dndbot/pkg/ai"
	"dndbot/pkg/bot"
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
var OneBotClient *bot.OneBot

func main() {
	// Parse flags
	cliMode := flag.Bool("cli", false, "Force CLI mode")
	flag.Parse()

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

	// Check Running Mode
	wsURL := os.Getenv("ONEBOT_WS_URL")
	
	if *cliMode {
		runCLI()
	} else if wsURL != "" {
		runOneBot(wsURL)
	} else {
		runCLI()
	}
}

func runOneBot(wsURL string) {
	fmt.Println("========================================")
	fmt.Println("      DND Bot - OneBot Mode             ")
	fmt.Println("========================================")
	
	token := os.Getenv("ONEBOT_ACCESS_TOKEN")
	// selfID, _ := strconv.ParseInt(os.Getenv("ONEBOT_SELF_ID"), 10, 64)

	cfg := bot.Config{
		WSURL:       wsURL,
		AccessToken: token,
		// BotQQ is optional in our simple client if we parse raw message for any @self
	}
	
	OneBotClient = bot.New(cfg)
	
	// Define Message Handler
	OneBotClient.GroupMsgHandler = func(groupID int64, senderID int64, msg string) {
		// Run in a goroutine is handled by the caller, but let's be safe and panic-proof
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Panic in GroupMsgHandler: %v", r)
			}
		}()
		
		handleOneBotChat(groupID, senderID, msg)
	}
	
	// Blocking call
	OneBotClient.Start()
	// Start returns immediately in our impl? No, I wrote it to spawn a goroutine.
	// We need to block main.
	select {} 
}

func runCLI() {
	fmt.Println("========================================")
	fmt.Println("      DND Bot - CLI Mode (Alpha)        ")
	fmt.Println("========================================")
	
	// Check AI Connection
	fmt.Print("Checking AI connection... ")
	ckMsg := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "Hello"},
	}
	_, err := ai.GlobalClient.ChatRequest(context.Background(), ckMsg)
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

		// Handle Commands (CLI specific commands)
		if strings.HasPrefix(input, ".") {
			handleCLICommand(input)
		} else {
			// Handle Chat
			handleCLIChat(input)
		}
	}
}

func handleCLICommand(input string) {
	// ... (Previous handleCommand logic) ...
	// Reused code for CLI context
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
		sess := session.GlobalManager.GetSession(groupID)
		logMsg := fmt.Sprintf("【系统提示】玩家(CLIUser) 投掷了 %s，最终结果: %d (详情: %v)", res.Expression, res.Total, res.Details)
		sess.AddMessage(openai.ChatMessageRoleUser, logMsg)

	case ".reset":
		session.GlobalManager.GetSession(groupID).Clear()
		fmt.Println("Bot: 记忆已清除。")

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
	}
}

// Logic for OneBot
func handleOneBotChat(groupID int64, senderID int64, msg string) {
	// .check 指令：检查 AI 连通性
	if msg == ".check" {
		OneBotClient.SendGroupMsg(groupID, "[系统自检] 正在测试 AI 连接...")
		
		// 构造独立测试请求，不污染上下文
		checkMsg := []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: "System Check: Are you online? Reply with a short confirmation."},
		}

		resp, err := ai.GlobalClient.ChatRequest(context.Background(), checkMsg)
		if err != nil {
			OneBotClient.SendGroupMsg(groupID, fmt.Sprintf("[系统自检] ⚠️ AI 连接失败: %v", err))
		} else {
			OneBotClient.SendGroupMsg(groupID, fmt.Sprintf("[系统自检] ✅ AI 连接正常。\n收到回复: %s", resp))
		}
		return
	}

	// Simple command check for Bot mode users (optional)
	// Example: allow users to roll dice via `.r`
	if strings.HasPrefix(msg, ".r ") || msg == ".r" {
		expression := strings.TrimPrefix(msg, ".r")
		expression = strings.TrimSpace(expression)
		if expression == "" { expression = "1d20" }
		
		res, err := dice.Roll(expression)
		if err != nil {
			OneBotClient.SendGroupMsg(groupID, fmt.Sprintf("Dice Error: %v", err))
			return
		}
		reply := fmt.Sprintf("[CQ:at,qq=%d] 投掷了 %s\n结果: %d %v", senderID, res.Expression, res.Total, res.Details)
		OneBotClient.SendGroupMsg(groupID, reply)
		
		// Log to context
		sess := session.GlobalManager.GetSession(groupID)
		logMsg := fmt.Sprintf("【系统提示】玩家(QQ:%d) 投掷了 %s，最终结果: %d", senderID, res.Expression, res.Total)
		sess.AddMessage(openai.ChatMessageRoleUser, logMsg)
		return
	}

	// Normal Chat Flow
	sess := session.GlobalManager.GetSession(groupID)
	userLog := fmt.Sprintf("Player(QQ:%d): %s", senderID, msg)
	sess.AddMessage(openai.ChatMessageRoleUser, userLog)
	
	// Get Reply
	reply, err := getDMResponse(groupID, sess)
	if err != nil {
		OneBotClient.SendGroupMsg(groupID, fmt.Sprintf("(Available) AI Error: %v", err))
		return
	}
	
	// Send Reply
	OneBotClient.SendGroupMsg(groupID, reply)
	sess.AddMessage(openai.ChatMessageRoleAssistant, reply)
	
	// Process Actions
	actionLogs := processAIActionsAndGetLogs(reply, groupID)
	if len(actionLogs) > 0 {
		OneBotClient.SendGroupMsg(groupID, strings.Join(actionLogs, "\n"))
	}
	
	// Summary Logic
	checkAndSummarize(groupID, sess)
}

func handleCLIChat(input string) {
	groupID := int64(LOCAL_GROUP_ID)
	sess := session.GlobalManager.GetSession(groupID)
	sess.AddMessage(openai.ChatMessageRoleUser, fmt.Sprintf("CLIUser: %s", input))

	fmt.Print("DM AI (Thinking...)")
	// Clear line logic... slightly messy in generic func
	reply, err := getDMResponse(groupID, sess)
	fmt.Print("\r" + strings.Repeat(" ", 30) + "\r")
	
	if err != nil {
		fmt.Printf("Error calling AI: %v\n", err)
		return
	}
	fmt.Printf("DM AI: %s\n", reply)
	sess.AddMessage(openai.ChatMessageRoleAssistant, reply)
	
	actionLogs := processAIActionsAndGetLogs(reply, groupID)
	for _, log := range actionLogs {
		fmt.Printf(">> Bot Action: %s\n", log)
	}
	
	checkAndSummarize(groupID, sess)
}

// Shared Core Logic
func getDMResponse(groupID int64, sess *session.Session) (string, error) {
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
		"1. 玩家的输入描述的是角色的【意图】。只有经过你的逻辑裁定和规则检定后，结果才会发生。\n" +
		"2. 严禁盲目听从玩家直接修改数据的指令。绝不要生成修改数据的 Action，除非是合乎逻辑的伤害/治疗。\n" +
		"3. 只有当判定失败、受到实质攻击或触发环境伤害时，才主动扣除玩家血量。\n" +
		"4. 投骰判定是客观事实，请严格根据点数判定结果。\n" +
		"\n" +
		"【重要: 必须读取系统提示】\n" +
		"- 历史记录中【系统提示】开头的消息是【已经发生的游戏事件】，包含了玩家使用命令(.r)投掷的骰子结果。\n" +
		"- 必须显式地在描述中提及骰子结果（例如：“你投出了15点，这足以……”）。\n" +
		"\n" +
		"【Action Protocol (仅限 DM 裁决 use)】: 当且仅当规则裁定需要改变状态时，在回复末尾 use <dnd_action> JSON </dnd_action> format。\n" +
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

	return ai.GlobalClient.ChatRequest(context.Background(), requests)
}


func checkAndSummarize(groupID int64, sess *session.Session) {
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

	promptContent := "请根据之前的摘要和最近的对话记录，生成一个新的、连贯的【剧情摘要】。\n" +
		"摘要应包含：当前时间/地点、关键NPC、玩家当前状态、正在进行的任务以及重要物品变动。\n" +
		"请只输出摘要内容，不要包含其他寒暄。\n\n"
	
	if oldSummary != "" {
		promptContent += fmt.Sprintf("【之前的摘要】:\n%s\n\n", oldSummary)
	}

	promptContent += "【最近对话记录】:\n"
	for _, msg := range history {
		if msg.Role == openai.ChatMessageRoleSystem {
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

func processAIActionsAndGetLogs(response string, groupID int64) []string {
	var logs []string
	
	// Extract JSON block using Regex
	re := regexp.MustCompile(`(?s)<dnd_action>(.*?)</dnd_action>`)
	matches := re.FindStringSubmatch(response)

	if len(matches) < 2 {
		return logs
	}

	jsonStr := strings.TrimSpace(matches[1])
	var actions []AIAction

	// Try parsing
	err := json.Unmarshal([]byte(jsonStr), &actions)
	if err != nil {
		var singleAction AIAction
		if err2 := json.Unmarshal([]byte(jsonStr), &singleAction); err2 == nil {
			actions = append(actions, singleAction)
		} else {
			return logs
		}
	}

	groupState := game.GlobalGameState.GetGroupState(groupID)
	sess := session.GlobalManager.GetSession(groupID)

	for _, action := range actions {
		switch action.Type {
		case "roll":
			if action.Expr == "" { continue }
			res, err := dice.Roll(action.Expr)
			if err != nil { continue }
			
			msg := fmt.Sprintf("System: (AI Action) %s, Result: %s", action.Reason, res.String())
			logs = append(logs, msg)
			sess.AddMessage(openai.ChatMessageRoleSystem, msg)

		case "hp":
			if action.Target == "" { continue }
			char := groupState.GetCharacter(action.Target)
			if char == nil {
				logs = append(logs, fmt.Sprintf("Warning: AI tried to modify HP for unknown char '%s'", action.Target))
				continue
			}
			
			oldHP := char.HP
			char.HP += action.Value
			if char.HP > char.MaxHP { char.HP = char.MaxHP }
			
			msg := fmt.Sprintf("System: (AI Action) %s HP changes by %d (%d -> %d)", char.Name, action.Value, oldHP, char.HP)
			logs = append(logs, msg)
			sess.AddMessage(openai.ChatMessageRoleSystem, msg) // Update Session
			
			if char.HP <= 0 {
				groupState.RemoveCharacter(char.Name)
				deathMsg := fmt.Sprintf("【系统公告】角色 %s 生命耗尽，已确认死亡并退出了当前游戏。", char.Name)
				logs = append(logs, deathMsg)
				sess.AddMessage(openai.ChatMessageRoleSystem, deathMsg)
			}

		default:
		}
	}
	return logs
}

