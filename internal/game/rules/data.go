package rules

// ValidAbilities 定义支持的基础属性缩写。
var ValidAbilities = map[string]struct{}{
	"str": {},
	"dex": {},
	"con": {},
	"int": {},
	"wis": {},
	"cha": {},
}

// SkillAbilityMap 定义技能与属性的最小映射关系。
var SkillAbilityMap = map[string]string{
	"athletics":       "str",
	"acrobatics":      "dex",
	"sleight_of_hand": "dex",
	"stealth":         "dex",
	"arcana":          "int",
	"history":         "int",
	"investigation":   "int",
	"nature":          "int",
	"religion":        "int",
	"animal_handling": "wis",
	"insight":         "wis",
	"medicine":        "wis",
	"perception":      "wis",
	"survival":        "wis",
	"deception":       "cha",
	"intimidation":    "cha",
	"performance":     "cha",
	"persuasion":      "cha",
}
