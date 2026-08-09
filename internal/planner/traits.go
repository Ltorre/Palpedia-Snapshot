package planner

import (
	"sort"
	"strings"
)

// TraitTier is the in-game quality tier used by the planner filters.
type TraitTier int

const (
	Unknown TraitTier = iota
	Gold
	Diamond
)

type traitInfo struct {
	name   string
	effect string
	tier   TraitTier
}

// traits translates current Palworld passive IDs to their player-facing names
// and concise in-game effects. IDs outside the catalog deliberately remain visible.
var traits = map[string]traitInfo{
	"Alien":                                {name: "Otherworldly Cells", effect: "Attack +10% Fire damage reduction 15% Lightning damage reduction 15%", tier: Unknown},
	"AutoHPRegeneRate_Passive":             {name: "Healing Coach", effect: "Player Auto Health Regeneration Rate +5%", tier: Unknown},
	"CoolTimeReduction_Down_1":             {name: "Easygoing", effect: "Active skill cooldown extension -15%", tier: Unknown},
	"CoolTimeReduction_Up_1":               {name: "Serenity", effect: "Active skill cooldown reduction 30% Attack +10%", tier: Gold},
	"CoolTimeReduction_Up_2":               {name: "Impatient", effect: "Active skill cooldown reduction 15%", tier: Unknown},
	"CraftSpeed_down1":                     {name: "Clumsy", effect: "Work Speed -10%", tier: Unknown},
	"CraftSpeed_down2":                     {name: "Slacker", effect: "Work Speed -30%", tier: Unknown},
	"CraftSpeed_up1":                       {name: "Serious", effect: "Work Speed +20%", tier: Unknown},
	"CraftSpeed_up2":                       {name: "Artisan", effect: "Work Speed +50%", tier: Gold},
	"CraftSpeed_up3":                       {name: "Remarkable Craftsmanship", effect: "Work Speed +75%", tier: Diamond},
	"Deffence_down1":                       {name: "Downtrodden", effect: "Defense -10%", tier: Unknown},
	"Deffence_down2":                       {name: "Brittle", effect: "Defense -20%", tier: Unknown},
	"Deffence_up1":                         {name: "Hard Skin", effect: "Defense +10%", tier: Unknown},
	"Deffence_up2":                         {name: "Burly Body", effect: "Defense +20% Immune to Flinch", tier: Gold},
	"Deffence_up2_2":                       {name: "Heavyweight", effect: "Defense +20% Immune to Knockback", tier: Unknown},
	"Deffence_up3":                         {name: "Diamond Body", effect: "Defense +30% Immune to Flinch Immune to Knockback", tier: Diamond},
	"ElementBoost_Aqua_1_PAL":              {name: "Hydromaniac", effect: "10% increase in Water attack damage.", tier: Unknown},
	"ElementBoost_Aqua_2_PAL":              {name: "Lord of the Sea", effect: "30% increase in Water attack damage.", tier: Gold},
	"ElementBoost_Dark_1_PAL":              {name: "Veil of Darkness", effect: "10% increase in Dark attack damage.", tier: Unknown},
	"ElementBoost_Dark_2_PAL":              {name: "Lord of the Underworld", effect: "30% increase in Dark attack damage.", tier: Gold},
	"ElementBoost_Dragon_1_PAL":            {name: "Blood of the Dragon", effect: "10% increase in Dragon attack damage.", tier: Unknown},
	"ElementBoost_Dragon_2_PAL":            {name: "Divine Dragon", effect: "30% increase in Dragon attack damage.", tier: Gold},
	"ElementBoost_Earth_1_PAL":             {name: "Power of Gaia", effect: "10% increase in Earth attack damage.", tier: Unknown},
	"ElementBoost_Earth_2_PAL":             {name: "Earth Emperor", effect: "30% increase in Earth attack damage.", tier: Gold},
	"ElementBoost_Fire_1_PAL":              {name: "Pyromaniac", effect: "10% increase in Fire attack damage.", tier: Unknown},
	"ElementBoost_Fire_2_PAL":              {name: "Flame Emperor", effect: "30% increase in Fire attack damage.", tier: Gold},
	"ElementBoost_Ice_1_PAL":               {name: "Coldblooded", effect: "10% increase in Ice attack damage.", tier: Unknown},
	"ElementBoost_Ice_2_PAL":               {name: "Ice Emperor", effect: "30% increase in Ice attack damage.", tier: Gold},
	"ElementBoost_Leaf_1_PAL":              {name: "Fragrant Foliage", effect: "10% increase in Grass attack damage.", tier: Unknown},
	"ElementBoost_Leaf_2_PAL":              {name: "Spirit Emperor", effect: "30% increase in Grass attack damage.", tier: Gold},
	"ElementBoost_Normal_1_PAL":            {name: "Spirit of Zen", effect: "10% increase in Neutral attack damage.", tier: Unknown},
	"ElementBoost_Normal_2_PAL":            {name: "Celestial Emperor", effect: "30% increase in Neutral attack damage.", tier: Gold},
	"ElementBoost_Thunder_1_PAL":           {name: "Capacitor", effect: "10% increase in Lightning attack damage.", tier: Unknown},
	"ElementBoost_Thunder_2_PAL":           {name: "Lord of Lightning", effect: "30% increase in Lightning attack damage.", tier: Gold},
	"ElementResist_Aqua_1_PAL":             {name: "Waterproof", effect: "10% decrease in incoming Water damage.", tier: Unknown},
	"ElementResist_Dark_1_PAL":             {name: "Cheery", effect: "10% decrease in incoming Dark damage.", tier: Unknown},
	"ElementResist_Dragon_1_PAL":           {name: "Dragonkiller", effect: "10% decrease in incoming Dragon damage.", tier: Unknown},
	"ElementResist_Earth_1_PAL":            {name: "Earthquake Resistant", effect: "10% decrease in incoming Earth damage.", tier: Unknown},
	"ElementResist_Fire_1_PAL":             {name: "Suntan Lover", effect: "10% decrease in incoming Fire damage.", tier: Unknown},
	"ElementResist_Ice_1_PAL":              {name: "Heated Body", effect: "10% decrease in incoming Ice damage.", tier: Unknown},
	"ElementResist_Leaf_1_PAL":             {name: "Botanical Barrier", effect: "10% decrease in incoming Grass damage.", tier: Unknown},
	"ElementResist_Normal_1_PAL":           {name: "Abnormal", effect: "10% decrease in incoming Neutral damage.", tier: Unknown},
	"ElementResist_Thunder_1_PAL":          {name: "Insulated Body", effect: "10% decrease in incoming Lightning damage.", tier: Unknown},
	"EternalFlame":                         {name: "Eternal Flame", effect: "30% increase to Fire attack damage. 30% increase to Lightning attack damage.", tier: Gold},
	"Invader":                              {name: "Invader", effect: "30% increase in Dark attack damage. 30% increase in Dragon attack damage.", tier: Diamond},
	"Legend":                               {name: "Legend", effect: "Attack +20% Defense +20% Movement Speed increases 20%", tier: Diamond},
	"MiniNushi":                            {name: "Whopper", effect: "5% increase to Water attack damage 5% increase to Ice attack damage 5% increase to defense.", tier: Unknown},
	"MoveSpeed_up_1":                       {name: "Nimble", effect: "10% increase to movement speed.", tier: Unknown},
	"MoveSpeed_up_2":                       {name: "Runner", effect: "20% increase to movement speed.", tier: Gold},
	"MoveSpeed_up_3":                       {name: "Swift", effect: "30% increase to movement speed.", tier: Diamond},
	"MutationPal_Babysitter":               {name: "Babysitter", effect: "While at a base, increases egg production speed by +30% and incubation speed by +30% for Pals assigned to a Breeding Farm.", tier: Diamond},
	"MutationPal_ExplosionResist":          {name: "Heavily Armored", effect: "Immune to Explosion Damage", tier: Unknown},
	"MutationPal_Immortal":                 {name: "Immortality", effect: "Life Steal +5% Pal Auto Health Regeneration Rate +100% Attack +15%", tier: Diamond},
	"MutationPal_Mutant":                   {name: "Idiosyncratic", effect: "Pal and Player Auto Health Regeneration Rate +50% Defense +25% Immune to Poison Damage Immune to Burn Damage", tier: Diamond},
	"NightOwl":                             {name: "Night Owl", effect: "Tends to nap through the day, due to being nocturnal.", tier: Unknown},
	"Nocturnal":                            {name: "Insomnia", effect: "Does not sleep and continues to work even at night.", tier: Unknown},
	"NonKilling":                           {name: "Mercy Hit", effect: "Pacifist. Will not reduce the target's Health below 1.", tier: Unknown},
	"Noukin":                               {name: "Musclehead", effect: "Attack +30%  Work Speed -50%", tier: Unknown},
	"Nushi":                                {name: "Lunker", effect: "20% increase to Water attack damage 20% increase to Ice attack damage 20% increase to defense.", tier: Diamond},
	"PAL_ALLAttack_down1":                  {name: "Coward", effect: "Attack -10%", tier: Unknown},
	"PAL_ALLAttack_down2":                  {name: "Pacifist", effect: "Attack -20%", tier: Unknown},
	"PAL_ALLAttack_up1":                    {name: "Brave", effect: "Attack +10%", tier: Unknown},
	"PAL_ALLAttack_up2":                    {name: "Ferocious", effect: "Attack +20%", tier: Gold},
	"PAL_ALLAttack_up3":                    {name: "Demon God", effect: "Attack +30%  Defense +5%", tier: Diamond},
	"PAL_conceited":                        {name: "Conceited", effect: "Work Speed +10%  Defense -10%", tier: Unknown},
	"PAL_CorporateSlave":                   {name: "Work Slave", effect: "Work Speed +30%  Attack -30%", tier: Unknown},
	"PAL_FullStomach_Down_1":               {name: "Dainty Eater", effect: "Hunger decreases +10.0% slower.", tier: Unknown},
	"PAL_FullStomach_Down_2":               {name: "Diet Lover", effect: "Hunger decreases +15.0% slower.", tier: Gold},
	"PAL_FullStomach_Down_3":               {name: "Mastery of Fasting", effect: "Hunger decreases +20.0% slower.", tier: Diamond},
	"PAL_FullStomach_Up_1":                 {name: "Glutton", effect: "Hunger decreases +10.0% faster.", tier: Unknown},
	"PAL_FullStomach_Up_2":                 {name: "Bottomless Stomach", effect: "Hunger decreases +15.0% faster.", tier: Unknown},
	"PAL_masochist":                        {name: "Masochist", effect: "Defense +15%  Attack -15%", tier: Unknown},
	"PAL_oraora":                           {name: "Aggressive", effect: "Attack +10%  Defense -10%", tier: Unknown},
	"PAL_rude":                             {name: "Hooligan", effect: "Attack +15%  Work Speed -10%", tier: Unknown},
	"PAL_sadist":                           {name: "Sadist", effect: "Attack +15%  Defense -15%", tier: Unknown},
	"PAL_Sanity_Down_1":                    {name: "Positive Thinker", effect: "SAN drops +10.0% slower.", tier: Unknown},
	"PAL_Sanity_Down_2":                    {name: "Workaholic", effect: "SAN drops +15.0% slower.", tier: Gold},
	"PAL_Sanity_Down_3":                    {name: "Heart of the Immovable King", effect: "SAN drops +20.0% slower.", tier: Diamond},
	"PAL_Sanity_Up_1":                      {name: "Unstable", effect: "SAN drops +10.0% faster.", tier: Unknown},
	"PAL_Sanity_Up_2":                      {name: "Destructive", effect: "SAN drops +15.0% faster.", tier: Unknown},
	"PlayerSP_DecreaseRate_Passive":        {name: "Wellness Watcher", effect: "Player Stamina Consumption - 5.0%", tier: Unknown},
	"Rare":                                 {name: "Lucky", effect: "Attack +15%  Defense +15% (None) Work Speed +20%", tier: Diamond},
	"ReloadSpeedUp_Passive":                {name: "Reload Master", effect: "Player Reload Speed +4%", tier: Unknown},
	"RideJumpCount_Increase1":              {name: "Lightfooted", effect: "Mounted Jump Count +1", tier: Unknown},
	"RideJumpCount_Increase2":              {name: "Skymarcher", effect: "Mounted Jump Count +2", tier: Unknown},
	"SalePrice_Down_1":                     {name: "Shabby", effect: "Decrease the value of items when sold by -10%", tier: Unknown},
	"SalePrice_Up_1":                       {name: "Noble", effect: "Increases the value of items when sold by +5%", tier: Gold},
	"SalePrice_Up_2":                       {name: "Fine Furs", effect: "Increases the value of items when sold by +3%", tier: Unknown},
	"Salvation":                            {name: "Savior", effect: "30% increase in Neutral attack damage. 30% increase in Grass attack damage.", tier: Diamond},
	"SelfDeathAddItemDrop_up_2":            {name: "Service-Minded", effect: "Your Dropped Items + 50%", tier: Unknown},
	"SelfDeathAddItemDrop_up_3":            {name: "Lavish Hospitality", effect: "Your Dropped Items + 100%", tier: Unknown},
	"Stamina_Down_1":                       {name: "Sickly", effect: "Max Stamina -25% *This effect is only valid for rideable pals.", tier: Unknown},
	"Stamina_Up_1":                         {name: "Infinite Stamina", effect: "Max stamina +50% *This effect is only valid for rideable pals.", tier: Gold},
	"Stamina_Up_2":                         {name: "Fit as a Fiddle", effect: "Max stamina +25% *This effect is only valid for rideable pals.", tier: Unknown},
	"Stamina_Up_3":                         {name: "Eternal Engine", effect: "Max stamina +75% *This effect is only valid for rideable pals.", tier: Diamond},
	"SwimSpeed_up_1":                       {name: "Sleek Stroke", effect: "30% increase movement speed on water.", tier: Unknown},
	"SwimSpeed_up_2":                       {name: "Ace Swimmer", effect: "40% increase movement speed on water.", tier: Gold},
	"SwimSpeed_up_3":                       {name: "King of the Waves", effect: "50% increase movement speed on water.", tier: Diamond},
	"Test_PalEgg_HatchingSpeed_Up":         {name: "Philanthropist", effect: "When assigned to a Breeding Farm, breeding speed is increased by 100%.", tier: Gold},
	"TrainerATK_UP_1":                      {name: "Vanguard", effect: "10% increase in Player Attack.", tier: Gold},
	"TrainerDEF_UP_1":                      {name: "Stronghold Strategist", effect: "10% increase in Player Defense.", tier: Gold},
	"TrainerLogging_up1":                   {name: "Logging Foreman", effect: "25% increase in Player Logging Efficiency.", tier: Gold},
	"TrainerMining_up1":                    {name: "Mine Foreman", effect: "25% increase in Player Mining Efficiency.", tier: Gold},
	"TrainerWorkSpeed_UP_1":                {name: "Motivational Leader", effect: "25% increase in Player Work Speed.", tier: Gold},
	"Vampire":                              {name: "Vampiric", effect: "Absorbs a portion of the damage dealt to restore Health. Does not sleep at night and continues to work.", tier: Diamond},
	"Witch":                                {name: "Siren of the Void", effect: "30% increase in Dark attack damage. 30% increase in Ice attack damage.", tier: Diamond},
	"WorkSuitabilityAddRank_MonsterFarm_1": {name: "Farmhand", effect: "Farming's Work Suitability +1", tier: Unknown},
	"WorkSuitabilityAddRank_MonsterFarm_2": {name: "Ranch Master", effect: "Farming's Work Suitability +2", tier: Diamond},
	"WorldTree_ATK":                        {name: "Twin-Edged Holy Blade", effect: "Attack +50% Defense -30% World Tree resources will not vanish when approached.", tier: Unknown},
	"WorldTree_ATK_DEF":                    {name: "God of Destruction", effect: "Attack +40% Defense +20% Max Health -50% World Tree resources will not vanish when approached.", tier: Unknown},
	"WorldTree_CraftSpeed":                 {name: "Demon\u2019s Hand", effect: "Work Speed +90 % SAN dreceases +15.0% faster World Tree harvestables won't vanish when approached.", tier: Unknown},
	"WorldTree_DEF":                        {name: "Sanctified Meat Shield", effect: "Defense +50% Attack -30% World Tree resources will not vanish when approached.", tier: Unknown},
	"WorldTree_FullStomach":                {name: "World Tree Seedbed", effect: "Decrease Hunger depletion rate by +50.0% HP -20 % World Tree resources will not vanish when approached.", tier: Unknown},
	"WorldTree_MoveSpeed":                  {name: "Dimensional Leap", effect: "Movement Speed +50 % Increases Hunger depletion rate by +15.0% World Tree resources will not vanish when approached.", tier: Unknown},
	"WorldTree_Sanity":                     {name: "Hermit Sage", effect: "SAN depletion rate - 50.0% Work Speed -20 % World Tree resources will not vanish when approached.", tier: Unknown},
}

// BreedingSpeedHelper is an owned Pal whose passive can speed up a long breeding route.
type BreedingSpeedHelper struct {
	Pal     Pal
	TraitID string
}

// BreedingSpeedHelpers finds owned Pals with base-breeding speed passives.
func BreedingSpeedHelpers(pals []Pal) []BreedingSpeedHelper {
	helpers := make([]BreedingSpeedHelper, 0)
	for _, pal := range pals {
		for _, trait := range pal.Traits {
			if trait == "Test_PalEgg_HatchingSpeed_Up" || trait == "MutationPal_Babysitter" {
				helpers = append(helpers, BreedingSpeedHelper{Pal: pal, TraitID: trait})
			}
		}
	}
	sort.Slice(helpers, func(i, j int) bool {
		if helpers[i].TraitID != helpers[j].TraitID {
			return helpers[i].TraitID == "MutationPal_Babysitter"
		}
		if helpers[i].Pal.CharacterID != helpers[j].Pal.CharacterID {
			return helpers[i].Pal.CharacterID < helpers[j].Pal.CharacterID
		}
		return helpers[i].Pal.InstanceID < helpers[j].Pal.InstanceID
	})
	return helpers
}

// Tier returns Gold for rank-3 and Diamond for rank-4 passive traits.
func Tier(id string) TraitTier {
	if info, ok := traits[id]; ok {
		return info.tier
	}
	return Unknown
}

// TraitName translates a raw save-game trait ID when the catalog knows it.
func TraitName(id string) string {
	if info, ok := traits[id]; ok {
		return info.name
	}
	return id
}

// TraitEffect returns the concise in-game effect, if known for a save-game trait ID.
func TraitEffect(id string) string {
	if info, ok := traits[id]; ok {
		return info.effect
	}
	return ""
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
