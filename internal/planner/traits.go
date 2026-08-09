package planner

import "strings"

// TraitTier is the in-game quality tier used by the planner filters.
type TraitTier int

const (
	Unknown TraitTier = iota
	Gold
	Diamond
)

type traitInfo struct {
	name string
	tier TraitTier
}

var traits = map[string]traitInfo{
	"SwimSpeed_up_2": {"Ace Swimmer", Gold}, "CraftSpeed_up2": {"Artisan", Gold}, "Deffence_up2": {"Burly Body", Gold}, "ElementBoost_Normal_2_PAL": {"Celestial Emperor", Gold}, "PAL_FullStomach_Down_2": {"Diet Lover", Gold}, "ElementBoost_Dragon_2_PAL": {"Divine Dragon", Gold}, "ElementBoost_Earth_2_PAL": {"Earth Emperor", Gold}, "EternalFlame": {"Eternal Flame", Gold}, "PAL_ALLAttack_up2": {"Ferocious", Gold}, "ElementBoost_Fire_2_PAL": {"Flame Emperor", Gold}, "ElementBoost_Ice_2_PAL": {"Ice Emperor", Gold}, "Stamina_Up_1": {"Infinite Stamina", Gold}, "TrainerLogging_up1": {"Logging Foreman", Gold}, "ElementBoost_Thunder_2_PAL": {"Lord of Lightning", Gold}, "ElementBoost_Aqua_2_PAL": {"Lord of the Sea", Gold}, "ElementBoost_Dark_2_PAL": {"Lord of the Underworld", Gold}, "TrainerMining_up1": {"Mine Foreman", Gold}, "TrainerWorkSpeed_UP_1": {"Motivational Leader", Gold}, "SalePrice_Up_1": {"Noble", Gold}, "Test_PalEgg_HatchingSpeed_Up": {"Philanthropist", Gold}, "MoveSpeed_up_2": {"Runner", Gold}, "CoolTimeReduction_Up_1": {"Serenity", Gold}, "ElementBoost_Leaf_2_PAL": {"Spirit Emperor", Gold}, "TrainerDEF_UP_1": {"Stronghold Strategist", Gold}, "TrainerATK_UP_1": {"Vanguard", Gold}, "PAL_Sanity_Down_2": {"Workaholic", Gold},
	"PAL_ALLAttack_up3": {"Demon God", Diamond}, "Deffence_up3": {"Diamond Body", Diamond}, "Stamina_Up_3": {"Eternal Engine", Diamond}, "PAL_Sanity_Down_3": {"Heart of the Immovable King", Diamond}, "MutationPal_Mutant": {"Idiosyncratic", Diamond}, "MutationPal_Immortal": {"Immortality", Diamond}, "Invader": {"Invader", Diamond}, "SwimSpeed_up_3": {"King of the Waves", Diamond}, "Legend": {"Legend", Diamond}, "Rare": {"Lucky", Diamond}, "Nushi": {"Lunker", Diamond}, "PAL_FullStomach_Down_3": {"Mastery of Fasting", Diamond}, "WorkSuitabilityAddRank_MonsterFarm_2": {"Ranch Master", Diamond}, "CraftSpeed_up3": {"Remarkable Craftsmanship", Diamond}, "Salvation": {"Savior", Diamond}, "Witch": {"Siren of the Void", Diamond}, "MoveSpeed_up_3": {"Swift", Diamond}, "Vampire": {"Vampiric", Diamond},
}

// Tier returns Gold for rank-3 and Diamond for rank-4 passive traits.
func Tier(id string) TraitTier {
	if info, ok := traits[id]; ok {
		return info.tier
	}
	return Unknown
}

// TraitName translates a raw save-game trait ID when this compact catalog knows it.
func TraitName(id string) string {
	if info, ok := traits[id]; ok {
		return info.name
	}
	return id
}

// TraitSummary produces a readable trait list while keeping unknown game IDs visible.
func TraitSummary(ids []string) string {
	labels := make([]string, 0, len(ids))
	for _, id := range ids {
		label := TraitName(id)
		switch Tier(id) {
		case Gold:
			label += " [gold]"
		case Diamond:
			label += " [diamond]"
		}
		labels = append(labels, label)
	}
	return strings.Join(labels, ", ")
}
