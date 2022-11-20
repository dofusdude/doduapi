package gen

type SearchIndexedMonster struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Race      string `json:"race"`
	SuperRace string `json:"super_race"`
	IsBoss    bool   `json:"is_boss"`
}

type SearchIndexedItem struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SuperType   string `json:"super_type"`
	TypeName    string `json:"type_name"`
	Level       int    `json:"level"`
}

type SearchIndexedMount struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	FamilyName string `json:"family_name"`
}

type SearchIndexedSet struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"highest_equipment_level"`
}

type EffectConditionDbEntry struct {
	Id   int
	Name string
}

type MappedMultilangCondition struct {
	Element   string            `json:"element"`
	ElementId int               `json:"element_id"`
	Operator  string            `json:"operator"`
	Value     int               `json:"value"`
	Templated map[string]string `json:"templated"`
}

type MappedMultilangRecipe struct {
	ResultId int                          `json:"result_id"`
	Entries  []MappedMultilangRecipeEntry `json:"entries"`
}

type MappedMultilangRecipeEntry struct {
	ItemId   int `json:"item_id"`
	Quantity int `json:"quantity"`
	//ItemType map[string]string `json:"item_type"`
}

type MappedMultilangSetReverseLink struct {
	Id   int               `json:"id"`
	Name map[string]string `json:"name"`
}

type MappedMultilangSet struct {
	AnkamaId int                       `json:"ankama_id"`
	Name     map[string]string         `json:"name"`
	ItemIds  []int                     `json:"items"`
	Effects  [][]MappedMultilangEffect `json:"effects"`
	Level    int                       `json:"level"`
}

type MappedMultilangMount struct {
	AnkamaId   int                     `json:"ankama_id"`
	Name       map[string]string       `json:"name"`
	FamilyId   int                     `json:"family_id"`
	FamilyName map[string]string       `json:"family_name"`
	Effects    []MappedMultilangEffect `json:"effects"`
}

type MappedMultilangCharacteristic struct {
	Value map[string]string `json:"value"`
	Name  map[string]string `json:"name"`
}

type MappedMultilangEffect struct {
	Min              int               `json:"min"`
	Max              int               `json:"max"`
	Type             map[string]string `json:"type"`
	MinMaxIrrelevant int               `json:"min_max_irrelevant"`
	Templated        map[string]string `json:"templated"`
	ElementId        int               `json:"element_id"`
	IsMeta           bool              `json:"is_meta"`
}

type MappedMultilangItemType struct {
	Id          int               `json:"id"`
	Name        map[string]string `json:"name"`
	SuperTypeId int               `json:"superTypeId"`
	CategoryId  int               `json:"categoryId"`
}

/*

type MonsterGrade struct {
	Grade             int `json:"grade"`
	Level             int `json:"level"`
	LifePoints        int `json:"lifePoints"`
	ActionPoints      int `json:"actionPoints"`
	MovementPoints    int `json:"movementPoints"`
	GradeXp           int `json:"gradeXp"`
	EarthResistance   int `json:"earthResistance"`
	FireResistance    int `json:"fireResistance"`
	WaterResistance   int `json:"waterResistance"`
	AirResistance     int `json:"airResistance"`
	NeutralResistance int `json:"neutralResistance"`
}

type MonsterDrops struct {
	ObjectId             int     `json:"objectId"`
	PercentDropForGrade1 float64 `json:"percentDropForGrade1"`
	PercentDropForGrade2 float64 `json:"percentDropForGrade2"`
	PercentDropForGrade3 float64 `json:"percentDropForGrade3"`
	PercentDropForGrade4 float64 `json:"percentDropForGrade4"`
	PercentDropForGrade5 float64 `json:"percentDropForGrade5"`
	HasCriteria          bool    `json:"hasCriteria"`
}

type JSONGameMonster struct {
	Id                          int            `json:"id"`
	NameId                      int            `json:"nameId"`
	Race                        int            `json:"race"`
	IsBoss                      bool           `json:"isBoss"`
	Grades                      []MonsterGrade `json:"grades"`
	Drops                       []MonsterDrops `json:"drops"`
	TemporisDrops               []MonsterDrops `json:"temporisDrops"`
	Subareas                    []int          `json:"subareas"`
	FavoriteSubareaId           int            `json:"favoriteSubareaId"`
	IsMiniBoss                  bool           `json:"isMiniBoss"`
	IsQuestMonster              bool           `json:"isQuestMonster"`
	CanBePushed                 bool           `json:"canBePushed"`
	CanTackle                   bool           `json:"canTackle"`
	CanSwitchPos                bool           `json:"canSwitchPos"`
	CanUsePortal                bool           `json:"canUsePortal"`
	IncompatibleIdols           []int          `json:"incompatibleIdols"`
	AllIdolsDisabled            bool           `json:"allIdolsDisabled"`
	IncompatibleChallenges      []int          `json:"incompatibleChallenges"`
	AggressiveZoneSize          int            `json:"aggressiveZoneSize"`
	AggressiveLevelDiff         int            `json:"aggressiveLevelDiff"`
	AggressiveImmunityCriterion string         `json:"aggressiveImmunityCriterion"`
}
*/

type MappedMultilangMonsterSuperRace struct {
	Id   int               `json:"id"`
	Name map[string]string `json:"name"`
}

type MappedMultilangMonsterRace struct {
	Id                  int                             `json:"id"`
	Name                map[string]string               `json:"name"`
	SuperRace           MappedMultilangMonsterSuperRace `json:"superRace"`
	AggressiveZoneSize  int                             `json:"aggressiveZoneSize"`
	AggressiveLevelDiff int                             `json:"aggressiveLevelDiff"`
}

type MappedMultilangMonster struct {
	AnkamaId            int                             `json:"ankama_id"`
	Name                map[string]string               `json:"name"`
	Race                MappedMultilangMonsterRace      `json:"race"`
	IsBoss              bool                            `json:"isBoss"`
	IsMiniBoss          bool                            `json:"isMiniBoss"`
	IsQuestMonster      bool                            `json:"isQuestMonster"`
	CanBePushed         bool                            `json:"canBePushed"`
	CanTackle           bool                            `json:"canTackle"`
	CanSwitchPos        bool                            `json:"canSwitchPos"`
	CanUsePortal        bool                            `json:"canUsePortal"`
	AllIdolsDisabled    bool                            `json:"allIdolsDisabled"`
	Grades              []MonsterGrade                  `json:"grades"`
	Drops               []MappedMultilangMonsterDrops   `json:"drops"`
	SubAreas            []MappedMultilangMonsterSubArea `json:"subareas"`
	AggressiveZoneSize  int                             `json:"aggressiveZoneSize"`
	AggressiveLevelDiff int                             `json:"aggressiveLevelDiff"`
	UseRaceValues       bool                            `json:"useRaceValues"`
}

type MappedMultilangSuperArea struct {
	Id   int               `json:"id"`
	Name map[string]string `json:"name"`
}

type MappedMultilangArea struct {
	Id              int                      `json:"id"`
	Name            map[string]string        `json:"name"`
	SuperArea       MappedMultilangSuperArea `json:"superArea"`
	ContainHouses   bool                     `json:"containHouses"`
	ContainPaddocks bool                     `json:"containPaddocks"`
}

type MappedMultilangMonsterSubArea struct {
	MappedMultilangSubArea
	IsFavorite bool `json:"isFavorite"`
}

type MappedMultilangSubArea struct {
	Id                   int                 `json:"id"`
	Name                 map[string]string   `json:"name"`
	Area                 MappedMultilangArea `json:"area"`
	Level                int                 `json:"level"`
	IsConquestVillage    bool                `json:"isConquestVillage"`
	SubscriberOnly       bool                `json:"subscriberOnly"`
	MountAutoTripAllowed bool                `json:"mountAutoTripAllowed"`
}

type MappedMultilangDrops struct {
	AnkamaId int    `json:"ankamaId"`
	Type     string `json:"type"`
}

type MappedMultilangItem struct {
	AnkamaId               int                             `json:"ankama_id"`
	Type                   MappedMultilangItemType         `json:"type"`
	Description            map[string]string               `json:"description"`
	Name                   map[string]string               `json:"name"`
	Image                  string                          `json:"image"`
	Conditions             []MappedMultilangCondition      `json:"conditions"`
	Level                  int                             `json:"level"`
	UsedInRecipes          []int                           `json:"used_in_recipes"`
	Characteristics        []MappedMultilangCharacteristic `json:"characteristics"`
	Effects                []MappedMultilangEffect         `json:"effects"`
	DropMonsterIds         []int                           `json:"dropMonsterIds"`
	CriticalHitBonus       int                             `json:"criticalHitBonus"`
	TwoHanded              bool                            `json:"twoHanded"`
	MaxCastPerTurn         int                             `json:"maxCastPerTurn"`
	ApCost                 int                             `json:"apCost"`
	Range                  int                             `json:"range"`
	MinRange               int                             `json:"minRange"`
	CriticalHitProbability int                             `json:"criticalHitProbability"`
	Pods                   int                             `json:"pods"`
	IconId                 int                             `json:"iconId"`
	ParentSet              MappedMultilangSetReverseLink   `json:"parentSet"`
	HasParentSet           bool                            `json:"hasParentSet"`
	DroppedBy              []MappedMultilangDrops          `json:"dropped_by"`
}

type JSONGameSpellType struct {
	Id          int `json:"id"`
	LongNameId  int `json:"longNameId"`
	ShortNameId int `json:"shortNameId"`
}

type JSONGameSpell struct {
	Id            int   `json:"id"`
	NameId        int   `json:"nameId"`
	DescriptionId int   `json:"descriptionId"`
	TypeId        int   `json:"typeId"`
	Order         int   `json:"order"`
	IconId        int   `json:"iconId"`
	SpellLevels   []int `json:"spellLevels"`
}

type JSONLangDict struct {
	Texts    map[string]string `json:"texts"`    // "1": "Account- oder Abohandel",
	IdText   map[string]int    `json:"idText"`   // "790745": 27679,
	NameText map[string]int    `json:"nameText"` // "ui.chat.check0": 65984
}

type JSONGameRecipe struct {
	Id            int   `json:"resultId"`
	NameId        int   `json:"resultNameId"`
	TypeId        int   `json:"resultTypeId"`
	Level         int   `json:"resultLevel"`
	IngredientIds []int `json:"ingredientIds"`
	Quantities    []int `json:"quantities"`
	JobId         int   `json:"jobId"`
	SkillId       int   `json:"skillId"`
}

type LangDict struct {
	Texts    map[int]string
	IdText   map[int]int
	NameText map[string]int
}

type JSONGameBonus struct {
	Amount        int   `json:"amount"`
	Id            int   `json:"id"`
	CriterionsIds []int `json:"criterionsIds"`
	Type          int   `json:"type"`
}

type JSONGameAreaBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type JSONGameArea struct {
	Id              int                `json:"id"`
	NameId          int                `json:"nameId"`
	SuperAreaId     int                `json:"superAreaId"`
	ContainHouses   bool               `json:"containHouses"`
	ContainPaddocks bool               `json:"containPaddocks"`
	Bounds          JSONGameAreaBounds `json:"bounds"`
	WorldMapId      int                `json:"worldmapId"`
	HasWorldMap     bool               `json:"hasWorldMap"`
}

type JSONGameItemPossibleEffect struct {
	EffectId     int `json:"effectId"`
	MinimumValue int `json:"diceNum"`
	MaximumValue int `json:"diceSide"`
	Value        int `json:"value"`

	BaseEffectId  int `json:"baseEffectId"`
	EffectElement int `json:"effectElement"`
	Dispellable   int `json:"dispellable"`
	SpellId       int `json:"spellId"`
	Duration      int `json:"duration"`
}

type JSONGameSet struct {
	Id      int                            `json:"id"`
	ItemIds []int                          `json:"items"`
	NameId  int                            `json:"nameId"`
	Effects [][]JSONGameItemPossibleEffect `json:"effects"`
}

type JSONGameItemType struct {
	Id          int `json:"id"`
	NameId      int `json:"nameId"`
	SuperTypeId int `json:"superTypeId"`
	CategoryId  int `json:"categoryId"`
}

type JSONGameEffect struct {
	Id                       int  `json:"id"`
	DescriptionId            int  `json:"descriptionId"`
	IconId                   int  `json:"iconId"`
	Characteristic           int  `json:"characteristic"`
	Category                 int  `json:"category"`
	UseDice                  bool `json:"useDice"`
	Active                   bool `json:"active"`
	TheoreticalDescriptionId int  `json:"theoreticalDescriptionId"`
	BonusType                int  `json:"bonusType"` // -1,0,+1
	ElementId                int  `json:"elementId"`
}

type JSONGameItem struct {
	Id            int `json:"id"`
	TypeId        int `json:"typeId"`
	DescriptionId int `json:"descriptionId"`
	IconId        int `json:"iconId"`
	NameId        int `json:"nameId"`
	Level         int `json:"level"`

	PossibleEffects        []JSONGameItemPossibleEffect `json:"possibleEffects"`
	RecipeIds              []int                        `json:"recipeIds"`
	Pods                   int                          `json:"realWeight"`
	ParseEffects           bool                         `json:"useDice"`
	EvolutiveEffectIds     []int                        `json:"evolutiveEffectIds"`
	DropMonsterIds         []int                        `json:"dropMonsterIds"`
	ItemSetId              int                          `json:"itemSetId"`
	Criteria               string                       `json:"criteria"`
	CriticalHitBonus       int                          `json:"criticalHitBonus"`
	TwoHanded              bool                         `json:"twoHanded"`
	MaxCastPerTurn         int                          `json:"maxCastPerTurn"`
	ApCost                 int                          `json:"apCost"`
	Range                  int                          `json:"range"`
	MinRange               int                          `json:"minRange"`
	CriticalHitProbability int                          `json:"criticalHitProbability"`
}

type JSONGameMount struct {
	Id       int                          `json:"id"`
	FamilyId int                          `json:"familyId"`
	NameId   int                          `json:"nameId"`
	Effects  []JSONGameItemPossibleEffect `json:"effects"`
}

type JSONGameMountFamily struct {
	Id      int    `json:"id"`
	NameId  int    `json:"nameId"`
	HeadUri string `json:"headUri"`
}

type MonsterGrade struct {
	Grade             int `json:"grade"`
	Level             int `json:"level"`
	LifePoints        int `json:"lifePoints"`
	ActionPoints      int `json:"actionPoints"`
	MovementPoints    int `json:"movementPoints"`
	GradeXp           int `json:"gradeXp"`
	EarthResistance   int `json:"earthResistance"`
	FireResistance    int `json:"fireResistance"`
	WaterResistance   int `json:"waterResistance"`
	AirResistance     int `json:"airResistance"`
	NeutralResistance int `json:"neutralResistance"`
}

type MappedMultilangMonsterDrops struct {
	ItemId               int     `json:"item_id"`
	ItemType             string  `json:"item_type"`
	PercentDropForGrade1 float64 `json:"percent_drop_for_grade_1"`
	PercentDropForGrade2 float64 `json:"percent_drop_for_grade_2"`
	PercentDropForGrade3 float64 `json:"percent_drop_for_grade_3"`
	PercentDropForGrade4 float64 `json:"percent_drop_for_grade_4"`
	PercentDropForGrade5 float64 `json:"percent_drop_for_grade_5"`
	HasCriteria          bool    `json:"hasCriteria"`
}

type MonsterDrops struct {
	ObjectId             int     `json:"objectId"`
	PercentDropForGrade1 float64 `json:"percentDropForGrade1"`
	PercentDropForGrade2 float64 `json:"percentDropForGrade2"`
	PercentDropForGrade3 float64 `json:"percentDropForGrade3"`
	PercentDropForGrade4 float64 `json:"percentDropForGrade4"`
	PercentDropForGrade5 float64 `json:"percentDropForGrade5"`
	HasCriteria          bool    `json:"hasCriteria"`
}

type JSONGameMonster struct {
	Id                          int            `json:"id"`
	NameId                      int            `json:"nameId"`
	Race                        int            `json:"race"`
	IsBoss                      bool           `json:"isBoss"`
	Grades                      []MonsterGrade `json:"grades"`
	Drops                       []MonsterDrops `json:"drops"`
	TemporisDrops               []MonsterDrops `json:"temporisDrops"`
	Subareas                    []int          `json:"subareas"`
	FavoriteSubareaId           int            `json:"favoriteSubareaId"`
	IsMiniBoss                  bool           `json:"isMiniBoss"`
	IsQuestMonster              bool           `json:"isQuestMonster"`
	CanBePushed                 bool           `json:"canBePushed"`
	CanTackle                   bool           `json:"canTackle"`
	CanSwitchPos                bool           `json:"canSwitchPos"`
	CanUsePortal                bool           `json:"canUsePortal"`
	IncompatibleIdols           []int          `json:"incompatibleIdols"`
	AllIdolsDisabled            bool           `json:"allIdolsDisabled"`
	IncompatibleChallenges      []int          `json:"incompatibleChallenges"`
	AggressiveZoneSize          int            `json:"aggressiveZoneSize"`
	AggressiveLevelDiff         int            `json:"aggressiveLevelDiff"`
	AggressiveImmunityCriterion string         `json:"aggressiveImmunityCriterion"`
	UseRaceValues               bool           `json:"useRaceValues"`
}

type JSONGameMonsterRace struct {
	Id                          int    `json:"id"`
	NameId                      int    `json:"nameId"`
	SuperRaceId                 int    `json:"superRaceId"`
	AggressiveZoneSize          int    `json:"aggressiveZoneSize"`
	AggressiveLevelDiff         int    `json:"aggressiveLevelDiff"`
	AggressiveImmunityCriterion string `json:"aggressiveImmunityCriterion"`
	Monsters                    []int  `json:"monsters"`
}

type JSONGameMonsterSuperRace struct {
	Id     int `json:"id"`
	NameId int `json:"nameId"`
}

type JSONGameSubAreaBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type JSONGameSubArea struct {
	Id                   int                `json:"id"`
	NameId               int                `json:"nameId"`
	AreaId               int                `json:"areaId"`
	MapIds               []float64          `json:"mapIds"`
	Bounds               JSONGameAreaBounds `json:"bounds"`
	Shape                []int              `json:"shape"`
	WorldMapId           int                `json:"worldmapId"`
	Level                int                `json:"level"`
	IsConquestVillage    bool               `json:"isConquestVillage"`
	BasicAccountAllowed  bool               `json:"basicAccountAllowed"`
	DisplayOnWorldMap    bool               `json:"displayOnWorldMap"`
	MountAutoTripAllowed bool               `json:"mountAutoTripAllowed"`
	Monsters             []int              `json:"monsters"`
	Quests               [][]float64        `json:"quests"`
	NPCs                 [][]float64        `json:"npcs"`
	Harvestables         []int              `json:"harvestables"`
	AssociatedZaapMapId  int                `json:"associatedZaapMapId"`
}

type JSONGameSuperArea struct {
	Id          int  `json:"id"`
	NameId      int  `json:"nameId"`
	WorldMapId  int  `json:"worldmapId"`
	HasWorldMap bool `json:"hasWorldMap"`
}

type JSONGameNPC struct {
	Id             int     `json:"id"`
	NameId         int     `json:"nameId"`
	DialogMessages [][]int `json:"dialogMessages"`
	DialogReplies  [][]int `json:"dialogReplies"`
	Actions        []int   `json:"actions"`
}

type JSONGameClass struct {
	Id                         int     `json:"id"`
	ShortNameId                int     `json:"shortNameId"`
	LongNameId                 int     `json:"longNameId"`
	DescriptionId              int     `json:"descriptionId"`
	GameplayDescriptionId      int     `json:"gameplayDescriptionId"`
	GameplayClassDescriptionId int     `json:"gameplayClassDescriptionId"` // TODO: remove <title iconSpellId=\"12982\">
	StatsPointsForStrength     [][]int `json:"statsPointsForStrength"`
	StatsPointsForIntelligence [][]int `json:"statsPointsForIntelligence"`
	StatsPointsForChance       [][]int `json:"statsPointsForChance"`
	StatsPointsForAgility      [][]int `json:"statsPointsForAgility"`
	StatsPointsForVitality     [][]int `json:"statsPointsForVitality"`
	StatsPointsForWisdom       [][]int `json:"statsPointsForWisdom"`
	BreedSpellsId              []int   `json:"breedSpellsId"`
	Complexity                 int     `json:"complexity"`
}

type MappedMultilangClass struct {
	Id                         int                    `json:"id"`
	ShortName                  map[string]string      `json:"shortName"`
	LongName                   map[string]string      `json:"longName"`
	Description                map[string]string      `json:"description"`
	GameplayDescription        map[string]string      `json:"gameplayDescription"`
	GameplayClassDescription   map[string]string      `json:"gameplayClassDescription"`
	StatsPointsForStrength     [][]int                `json:"statsPointsForStrength"`
	StatsPointsForIntelligence [][]int                `json:"statsPointsForIntelligence"`
	StatsPointsForChance       [][]int                `json:"statsPointsForChance"`
	StatsPointsForAgility      [][]int                `json:"statsPointsForAgility"`
	StatsPointsForVitality     [][]int                `json:"statsPointsForVitality"`
	StatsPointsForWisdom       [][]int                `json:"statsPointsForWisdom"`
	Spells                     []MappedMultilangSpell `json:"spells"`
	Complexity                 int                    `json:"complexity"`
}

/*
"modifiableRange": true,
              "isLinear": false,
              "needsFreeCell": false,
              "aoeType": {
                "en": "1-cell diagonal cross",
                "fr": "Croix diagonale de 1 case",
                "de": "Diagonal-Kreuzförmiger Wirkungsbereich von 1 Feld",
                "es": "Cruz en diagonal de 1 casilla",
                "it": "Croce diagonale di 1 casella",
                "pt": "Cruz diagonal de 1 célula"
              },
              "spellRange": {
                "minRange": 1,
                "maxRange": 5
              },
              "normalEffects": {
                "modifiableEffect": [
                  {
                    "stat": "Neutral damage",
                    "minStat": 12,
                    "maxStat": 14
                  }
                ],
                "customEffect": {
                  "en": ["-2 AP (1 turn)"],
                  "fr": ["-2 PA (1 tour)"],
                  "de": ["-2 AP (1 runde)"],
                  "es": ["-2 PA (1 turno)"],
                  "it": ["-2 PA (1 turno)"],
                  "pt": ["-2 PA (1 turno)"]
                }
              },
              "criticalEffects": {
                "modifiableEffect": [
                  {
                    "stat": "Neutral damage",
                    "minStat": 14,
                    "maxStat": 16
                  }
                ],
                "customEffect": {
                  "en": ["-2 AP (1 turn)"],
                  "fr": ["-2 PA (1 tour)"],
                  "de": ["-2 AP (1 runde)"],
                  "es": ["-2 PA (1 turno)"],
                  "it": ["-2 PA (1 turno)"],
                  "pt": ["-2 PA (1 turno)"]
                }
              }
*/

type MappedSpellRange struct {
	MinRange int `json:"minRange"`
	MaxRange int `json:"maxRange"`
}

type MappedModifiableEffect struct {
	Stat    string `json:"stat"`
	MinStat int    `json:"minStat"`
	MaxStat int    `json:"maxStat"`
}

type MappedMultilangSpellEffect struct {
	Id               int               `json:"id"`
	Level            int               `json:"level"`
	Cooldown         *int              `json:"cooldown"`
	BaseCriticalRate *int              `json:"baseCriticalRate"`
	CastsPerPlayer   *int              `json:"castsPerPlayer"`
	CastsPerTurn     *int              `json:"castsPerTurn"`
	NeedLos          bool              `json:"needLos"`
	ModifiableRange  bool              `json:"modifiableRange"`
	IsLinear         bool              `json:"isLinear"`
	NeedsFreeCell    bool              `json:"needsFreeCell"`
	AoeType          map[string]string `json:"aoeType"`
	SpellRange       MappedSpellRange  `json:"spellRange"`
	NormalEffects    struct {
		ModifiableEffect []MappedMultilangEffect `json:"modifiableEffect"`
		CustomEffect     map[string][]string     `json:"customEffect"`
	} `json:"normalEffects"`
	CriticalEffects struct {
		ModifiableEffect []MappedMultilangEffect `json:"modifiableEffect"`
		CustomEffect     map[string][]string     `json:"customEffect"`
	}
}

type MappedMultilangSpell struct {
	Id          int                          `json:"id"`
	Name        map[string]string            `json:"name"`
	Description map[string]string            `json:"description"`
	ImageUrls   []string                     `json:"imageUrls"` // TODO render
	Effects     []MappedMultilangSpellEffect `json:"effects"`
}

type JSONGameSpellPair struct {
	Id int `json:"id"`
}

type JSONGameSpellBomb struct {
	Id int `json:"id"`
}
type JSONGameSpellState struct {
	Id int `json:"id"`
}

type JSONGameSpellEffect struct {
	JSONGameItemPossibleEffect
	Random float64 `json:"random"`
	Delay  int     `json:"delay"`
}

type JSONGameSpellLevel struct {
	Id                     int  `json:"id"`
	SpellId                int  `json:"spellId"`
	Grade                  int  `json:"grade"`
	SpellBreed             int  `json:"spellBreed"`
	ApCost                 int  `json:"apCost"`
	MinRange               int  `json:"minRange"`
	Range                  int  `json:"range"`
	CastInLine             bool `json:"castInLine"`
	CastInDiagonal         bool `json:"castInDiagonal"`
	CastTestLos            bool `json:"castTestLos"`
	CriticalHitProbability int  `json:"criticalHitProbability"`
	NeedFreeCell           bool `json:"needFreeCell"`
	NeedTakenCell          bool `json:"needTakenCell"`
	NeedFreeTrapCell       bool `json:"needFreeTrapCell"`
	RangeCanBeBoosted      bool `json:"rangeCanBeBoosted"`
	MaxStack               int  `json:"maxStack"`
	MaxCastPerTurn         int  `json:"maxCastPerTurn"`
	MaxCastPerTarget       int  `json:"maxCastPerTarget"`
	MinCastInterval        int  `json:"minCastInterval"`
	InitialCooldown        int  `json:"initialCooldown"`
	GlobalCooldown         int  `json:"globalCooldown"`
	MinPlayerLevel         int  `json:"minPlayerLevel"`
	Effects
}
type JSONGameSpellConversion struct {
	Id int `json:"id"`
}

type JSONGameData struct {
	Items             map[int]JSONGameItem
	Sets              map[int]JSONGameSet
	ItemTypes         map[int]JSONGameItemType
	Effects           map[int]JSONGameEffect
	Bonuses           map[int]JSONGameBonus
	Recipes           map[int]JSONGameRecipe
	Spells            map[int]JSONGameSpell
	SpellLevels       map[int]JSONGameSpellLevel
	SpellPairs        map[int]JSONGameSpellPair
	SpellBombs        map[int]JSONGameSpellBomb
	SpellStates       map[int]JSONGameSpellState
	SpellConversions  map[int]JSONGameSpellConversion
	SpellTypes        map[int]JSONGameSpellType
	Areas             map[int]JSONGameArea
	SuperAreas        map[int]JSONGameSuperArea
	SubAreas          map[int]JSONGameSubArea
	Mounts            map[int]JSONGameMount
	Classes           map[int]JSONGameClass
	MountFamilys      map[int]JSONGameMountFamily
	Npcs              map[int]JSONGameNPC
	Monsters          map[int]JSONGameMonster
	MonsterRaces      map[int]JSONGameMonsterRace
	MonsterSuperRaces map[int]JSONGameMonsterSuperRace
	ItemMonsterDrops  map[int][]int
}
