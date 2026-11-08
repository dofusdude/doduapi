package main

import (
	"fmt"

	"github.com/dofusdude/doduapi/database"
	"github.com/dofusdude/doduapi/utils"
	"github.com/hashicorp/go-memdb"
)

type APIZoneShape struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type APIZone struct {
	Raw                         string       `json:"raw"`
	Shape                       APIZoneShape `json:"shape"`
	Size                        *int         `json:"size,omitempty"`
	MinSize                     *int         `json:"min_size,omitempty"`
	CellIds                     []int        `json:"cell_ids"`
	DamageDecreaseStepPercent   int          `json:"damage_decrease_step_percent"`
	MaxDamageDecreaseApplyCount int          `json:"max_damage_decrease_apply_count"`
	ForcedDirection             int          `json:"forced_direction"`
	IncludeCarried              bool         `json:"include_carried"`
	StopAtTarget                bool         `json:"stop_at_target"`
	OnlyAffectInSightLine       bool         `json:"only_affect_in_sight_line"`
}

type APIWeaponCombat struct {
	Zone                APIZone `json:"zone"`
	CastInLine          bool    `json:"cast_in_line"`
	CastInDiagonal      bool    `json:"cast_in_diagonal"`
	RequiresLineOfSight bool    `json:"requires_line_of_sight"`
}

type weaponCombatRecord struct {
	AnkamaId int
	Combat   APIWeaponCombat
}

func getWeaponCombat(ankamaId int, txn *memdb.Txn) *APIWeaponCombat {
	raw, err := txn.First(fmt.Sprintf("%s-item_weapon_combat", utils.CurrentRedBlueVersionStr(database.Version.MemDb)), "id", ankamaId)
	if err != nil || raw == nil {
		return nil
	}
	combat := raw.(*weaponCombatRecord).Combat
	return &combat
}
