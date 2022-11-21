package update

import (
	"github.com/dofusdude/ankabuffer"
	"sync"
)

func DownloadItems(hashJson *ankabuffer.Manifest) {
	files := hashJson.Fragments["main"].Files

	var wg sync.WaitGroup

	var bonusCriterions HashFile
	bonusCriterions.Filename = "data/common/BonusesCriterions.d2o"
	bonusCriterions.FriendlyName = "data/tmp/bonus_criterions.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, bonusCriterions, "data", "d2o")
	}()

	var evolEffects HashFile
	evolEffects.Filename = "data/common/EvolutiveEffects.d2o"
	evolEffects.FriendlyName = "data/tmp/evol_effects.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, evolEffects, "data", "d2o")
	}()

	var createBoneOverrides HashFile
	createBoneOverrides.Filename = "data/common/CreatureBonesOverrides.d2o"
	createBoneOverrides.FriendlyName = "data/tmp/create_bone_overrides.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, createBoneOverrides, "data", "d2o")
	}()

	var creatureBoneTypes HashFile
	creatureBoneTypes.Filename = "data/common/CreatureBonesTypes.d2o"
	creatureBoneTypes.FriendlyName = "data/tmp/creature_bone_types.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, creatureBoneTypes, "data", "d2o")
	}()

	var charsCategories HashFile
	charsCategories.Filename = "data/common/CharacteristicCategories.d2o"
	charsCategories.FriendlyName = "data/tmp/chars_categories.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, charsCategories, "data", "d2o")
	}()

	var serverGameTypes HashFile
	serverGameTypes.Filename = "data/common/ServerGameTypes.d2o"
	serverGameTypes.FriendlyName = "data/tmp/server_game_types.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, serverGameTypes, "data", "d2o")
	}()

	var npcs HashFile
	npcs.Filename = "data/common/Npcs.d2o"
	npcs.FriendlyName = "data/tmp/npcs.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, npcs, "data", "d2o")
	}()

	var mountFamily HashFile
	mountFamily.Filename = "data/common/MountFamily.d2o"
	mountFamily.FriendlyName = "data/tmp/mount_family.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, mountFamily, "data", "d2o")
	}()

	var areas HashFile
	areas.Filename = "data/common/Areas.d2o"
	areas.FriendlyName = "data/tmp/areas.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, areas, "data", "d2o")
	}()

	var companions HashFile
	companions.Filename = "data/common/Companions.d2o"
	companions.FriendlyName = "data/tmp/companions.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, companions, "data", "d2o")
	}()

	var companionSpells HashFile
	companionSpells.Filename = "data/common/CompanionSpells.d2o"
	companionSpells.FriendlyName = "data/tmp/companion_spells.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, companionSpells, "data", "d2o")
	}()

	var companionChars HashFile
	companionChars.Filename = "data/common/CompanionCharacteristics.d2o"
	companionChars.FriendlyName = "data/tmp/companion_chars.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, companionChars, "data", "d2o")
	}()

	var monsters HashFile
	monsters.Filename = "data/common/Monsters.d2o"
	monsters.FriendlyName = "data/tmp/monsters.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, monsters, "data", "d2o")
	}()

	var monsterRaces HashFile
	monsterRaces.Filename = "data/common/MonsterRaces.d2o"
	monsterRaces.FriendlyName = "data/tmp/monster_races.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, monsterRaces, "data", "d2o")
	}()

	var almanax HashFile
	almanax.Filename = "data/common/AlmanaxCalendars.d2o"
	almanax.FriendlyName = "data/tmp/almanax.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, almanax, "data", "d2o")
	}()

	var idols HashFile
	idols.Filename = "data/common/Idols.d2o"
	idols.FriendlyName = "data/tmp/idols.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, idols, "data", "d2o")
	}()

	var mounts HashFile
	mounts.Filename = "data/common/Mounts.d2o"
	mounts.FriendlyName = "data/tmp/mounts.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, mounts, "data", "d2o")
	}()

	var breeds HashFile
	breeds.Filename = "data/common/Breeds.d2o"
	breeds.FriendlyName = "data/tmp/breeds.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, breeds, "data", "d2o")
	}()

	var spellTypes HashFile
	spellTypes.Filename = "data/common/SpellTypes.d2o"
	spellTypes.FriendlyName = "data/tmp/spell_types.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, spellTypes, "data", "d2o")
	}()

	var spells HashFile
	spells.Filename = "data/common/Spells.d2o"
	spells.FriendlyName = "data/tmp/spells.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, spells, "data", "d2o")
	}()

	var recipes HashFile
	recipes.Filename = "data/common/Recipes.d2o"
	recipes.FriendlyName = "data/tmp/recipes.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, recipes, "data", "d2o")
	}()

	var bonuses HashFile
	bonuses.Filename = "data/common/Bonuses.d2o"
	bonuses.FriendlyName = "data/tmp/bonuses.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, bonuses, "data", "d2o")
	}()

	var effects HashFile
	effects.Filename = "data/common/Effects.d2o"
	effects.FriendlyName = "data/tmp/effects.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, effects, "data", "d2o")
	}()

	// item types
	var itemTypes HashFile
	itemTypes.Filename = "data/common/ItemTypes.d2o"
	itemTypes.FriendlyName = "data/tmp/item_types.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, itemTypes, "data", "d2o")
	}()

	var itemSets HashFile
	itemSets.Filename = "data/common/ItemSets.d2o"
	itemSets.FriendlyName = "data/tmp/item_sets.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, itemSets, "data", "d2o")
	}()

	var items HashFile
	items.Filename = "data/common/Items.d2o"
	items.FriendlyName = "data/tmp/items.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, items, "data", "d2o")
	}()

	wg.Wait()
}
