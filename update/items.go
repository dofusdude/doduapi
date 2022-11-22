package update

import (
	"github.com/dofusdude/ankabuffer"
)

func DownloadItems(hashJson *ankabuffer.Manifest) {
	fileNames := []HashFile{
		{Filename: "data/common/Items.d2o", FriendlyName: "data/tmp/items.d2o"},
		{Filename: "data/common/ItemTypes.d2o", FriendlyName: "data/tmp/item_types.d2o"},
		{Filename: "data/common/ItemSets.d2o", FriendlyName: "data/tmp/item_sets.d2o"},
		{Filename: "data/common/Effects.d2o", FriendlyName: "data/tmp/effects.d2o"},
		{Filename: "data/common/Bonuses.d2o", FriendlyName: "data/tmp/bonuses.d2o"},
		{Filename: "data/common/Recipes.d2o", FriendlyName: "data/tmp/recipes.d2o"},
		{Filename: "data/common/Spells.d2o", FriendlyName: "data/tmp/spells.d2o"},
		{Filename: "data/common/SpellTypes.d2o", FriendlyName: "data/tmp/spell_types.d2o"},
		{Filename: "data/common/Breeds.d2o", FriendlyName: "data/tmp/breeds.d2o"},
		{Filename: "data/common/Mounts.d2o", FriendlyName: "data/tmp/mounts.d2o"},
		{Filename: "data/common/Idols.d2o", FriendlyName: "data/tmp/idols.d2o"},
		{Filename: "data/common/AlmanaxCalendars.d2o", FriendlyName: "data/tmp/almanax.d2o"},
		{Filename: "data/common/MonsterRaces.d2o", FriendlyName: "data/tmp/monster_races.d2o"},
		{Filename: "data/common/Monsters.d2o", FriendlyName: "data/tmp/monsters.d2o"},
		{Filename: "data/common/CompanionCharacteristics.d2o", FriendlyName: "data/tmp/companion_chars.d2o"},
		{Filename: "data/common/CompanionSpells.d2o", FriendlyName: "data/tmp/companion_spells.d2o"},
		{Filename: "data/common/Companions.d2o", FriendlyName: "data/tmp/companions.d2o"},
		{Filename: "data/common/Areas.d2o", FriendlyName: "data/tmp/areas.d2o"},
		{Filename: "data/common/MountFamily.d2o", FriendlyName: "data/tmp/mount_family.d2o"},
		{Filename: "data/common/Npcs.d2o", FriendlyName: "data/tmp/npcs.d2o"},
		{Filename: "data/common/ServerGameTypes.d2o", FriendlyName: "data/tmp/server_game_types.d2o"},
		{Filename: "data/common/CharacteristicCategories.d2o", FriendlyName: "data/tmp/chars_categories.d2o"},
		{Filename: "data/common/CreatureBonesTypes.d2o", FriendlyName: "data/tmp/creature_bone_types.d2o"},
		{Filename: "data/common/CreatureBonesOverrides.d2o", FriendlyName: "data/tmp/create_bone_overrides.d2o"},
		{Filename: "data/common/EvolutiveEffects.d2o", FriendlyName: "data/tmp/evol_effects.d2o"},
		{Filename: "data/common/BonusesCriterions.d2o", FriendlyName: "data/tmp/bonus_criterions.d2o"},
	}

	DownloadUnpackFiles(hashJson, "main", fileNames, "data", true)
}
