package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/dodumap"
	_ "github.com/mattn/go-sqlite3"
)

var repositoryMutex = sync.Mutex{}
var DatabaseName = "almanax.db"

type Repository struct {
	Db  *sql.DB
	ctx context.Context
}

func NewDatabaseRepository(ctx context.Context, workdir string) *Repository {
	repo := Repository{}
	repo.Init(ctx, workdir)
	return &repo
}

func (r *Repository) Init(ctx context.Context, workdir string) error {
	dbpath := path.Join(workdir, DatabaseName)
	// check if the file exists
	_, err := os.Stat(dbpath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("Database does not exist, creating")
			file, err := os.Create(dbpath)
			if err != nil {
				return err
			}
			file.Close()
		} else {
			return err
		}
	}

	sqliteDatabase, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		return err
	}
	r.Db = sqliteDatabase
	r.ctx = ctx

	return nil
}

func (r *Repository) Deinit() {
	r.Db.Close()
	r.Db = nil
}

func (r *Repository) GetAlmanaxByDateRangeAndNameID(from, to, nameID string) ([]MappedAlmanax, error) {
	query := `
		SELECT
			a.id, a.bonus_id, a.tribute_id, a.date, a.reward_kamas, a.created_at, a.updated_at, a.deleted_at,
			b.id, b.bonus_type_id, b.description_en, b.description_fr, b.description_es, b.description_de, b.description_pt,
			bt.id, bt.name_id, bt.name_en, bt.name_fr, bt.name_es, bt.name_de, bt.name_pt,
			t.id, t.item_name_en, t.item_name_fr, t.item_name_es, t.item_name_de, t.item_name_pt,
			t.item_ankama_id, t.item_category_id, t.item_doduapi_uri, t.quantity
		FROM almanax AS a
		JOIN bonus AS b ON a.bonus_id = b.id
		JOIN bonus_types AS bt ON b.bonus_type_id = bt.id
		JOIN tribute AS t ON a.tribute_id = t.id
		WHERE a.date >= ? AND a.date <= ? AND bt.name_id = ? AND a.deleted_at IS NULL
		ORDER BY a.date ASC`

	rows, err := r.Db.Query(query, from, to, nameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MappedAlmanax

	for rows.Next() {
		var denorm MappedAlmanax
		var deletedAt sql.NullTime

		err := rows.Scan(
			&denorm.Almanax.ID, &denorm.Almanax.BonusID, &denorm.Almanax.TributeID, &denorm.Almanax.Date,
			&denorm.Almanax.RewardKamas, &denorm.Almanax.CreatedAt, &denorm.Almanax.UpdatedAt, &deletedAt,
			&denorm.Bonus.ID, &denorm.Bonus.BonusTypeID, &denorm.Bonus.DescriptionEn, &denorm.Bonus.DescriptionFr,
			&denorm.Bonus.DescriptionEs, &denorm.Bonus.DescriptionDe, &denorm.Bonus.DescriptionPt,
			&denorm.BonusType.ID, &denorm.BonusType.NameID, &denorm.BonusType.NameEn, &denorm.BonusType.NameFr,
			&denorm.BonusType.NameEs, &denorm.BonusType.NameDe, &denorm.BonusType.NamePt,
			&denorm.Tribute.ID, &denorm.Tribute.ItemNameEn, &denorm.Tribute.ItemNameFr, &denorm.Tribute.ItemNameEs, &denorm.Tribute.ItemNameDe, &denorm.Tribute.ItemNamePt,
			&denorm.Tribute.ItemAnkamaID, &denorm.Tribute.ItemCategoryId,
			&denorm.Tribute.ItemDoduapiUri, &denorm.Tribute.Quantity)

		if err != nil {
			return nil, err
		}
		if deletedAt.Valid {
			denorm.Almanax.DeletedAt = &deletedAt.Time
		}

		result = append(result, denorm)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) GetAlmanaxByDateRange(from, to string) ([]MappedAlmanax, error) {
	query := `
		SELECT
			a.id, a.bonus_id, a.tribute_id, a.date, a.reward_kamas, a.created_at, a.updated_at, a.deleted_at,
			b.id, b.bonus_type_id, b.description_en, b.description_fr, b.description_es, b.description_de, b.description_pt,
			bt.id, bt.name_id, bt.name_en, bt.name_fr, bt.name_es, bt.name_de, bt.name_pt,
			t.id, t.item_name_en, t.item_name_fr, t.item_name_es, t.item_name_de, t.item_name_pt,
			t.item_ankama_id, t.item_category_id, t.item_doduapi_uri, t.quantity
		FROM almanax AS a
		JOIN bonus AS b ON a.bonus_id = b.id
		JOIN bonus_types AS bt ON b.bonus_type_id = bt.id
		JOIN tribute AS t ON a.tribute_id = t.id
		WHERE a.date >= ? AND a.date <= ? AND a.deleted_at IS NULL
		ORDER BY a.date ASC`

	rows, err := r.Db.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []MappedAlmanax

	for rows.Next() {
		var denorm MappedAlmanax
		var deletedAt sql.NullTime

		err := rows.Scan(
			&denorm.Almanax.ID, &denorm.Almanax.BonusID, &denorm.Almanax.TributeID, &denorm.Almanax.Date,
			&denorm.Almanax.RewardKamas, &denorm.Almanax.CreatedAt, &denorm.Almanax.UpdatedAt, &deletedAt,
			&denorm.Bonus.ID, &denorm.Bonus.BonusTypeID, &denorm.Bonus.DescriptionEn, &denorm.Bonus.DescriptionFr,
			&denorm.Bonus.DescriptionEs, &denorm.Bonus.DescriptionDe, &denorm.Bonus.DescriptionPt,
			&denorm.BonusType.ID, &denorm.BonusType.NameID, &denorm.BonusType.NameEn, &denorm.BonusType.NameFr,
			&denorm.BonusType.NameEs, &denorm.BonusType.NameDe, &denorm.BonusType.NamePt,
			&denorm.Tribute.ID, &denorm.Tribute.ItemNameEn, &denorm.Tribute.ItemNameFr, &denorm.Tribute.ItemNameEs, &denorm.Tribute.ItemNameDe, &denorm.Tribute.ItemNamePt,
			&denorm.Tribute.ItemAnkamaID, &denorm.Tribute.ItemCategoryId,
			&denorm.Tribute.ItemDoduapiUri, &denorm.Tribute.Quantity)

		if err != nil {
			return nil, err
		}
		if deletedAt.Valid {
			denorm.Almanax.DeletedAt = &deletedAt.Time
		}

		result = append(result, denorm)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *Repository) CreateBonus(bonus *Bonus) (int64, error) {
	query := `INSERT INTO bonus (bonus_type_id, description_en, description_fr, description_es, description_de, description_pt, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`
	result, err := r.Db.Exec(query, bonus.BonusTypeID, bonus.DescriptionEn, bonus.DescriptionFr, bonus.DescriptionEs,
		bonus.DescriptionDe, bonus.DescriptionPt)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (r *Repository) CreateTribute(tribute *Tribute) (int64, error) {
	query := `INSERT INTO tribute (item_name_en, item_name_fr, item_name_es, item_name_de, item_name_pt, item_ankama_id, item_category_id, item_doduapi_uri, quantity, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`
	result, err := r.Db.Exec(query, tribute.ItemNameEn, tribute.ItemNameFr, tribute.ItemNameEs, tribute.ItemNameDe,
		tribute.ItemNamePt, tribute.ItemAnkamaID, tribute.ItemCategoryId, tribute.ItemDoduapiUri, tribute.Quantity)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (r *Repository) CreateOrUpdate(date string, almanax *dodumap.MappedMultilangNPCAlmanaxUnity) (int64, error) {
	bonusType := enNameToId(almanax.Bonus["en"])
	query := `SELECT id FROM bonus_types WHERE name_id = ?`
	var bonusTypeID int64
	err := r.Db.QueryRow(query, bonusType).Scan(&bonusTypeID)
	if err == sql.ErrNoRows {
		bonusTypeID, err = r.CreateBonusType(&BonusType{
			NameID: bonusType,
			NameEn: almanax.Bonus["en"],
			NameFr: almanax.Bonus["fr"],
			NameEs: almanax.Bonus["es"],
			NameDe: almanax.Bonus["de"],
			NamePt: almanax.Bonus["pt"],
		})
		if err != nil {
			return -1, err
		}
	}

	query = `SELECT id FROM bonus WHERE description_en = ?`
	var bonusID int64
	err = r.Db.QueryRow(query, almanax.Bonus["en"]).Scan(&bonusID)
	if err == sql.ErrNoRows {
		bonusID, err = r.CreateBonus(&Bonus{
			BonusTypeID:   bonusTypeID,
			DescriptionEn: almanax.Bonus["en"],
			DescriptionFr: almanax.Bonus["fr"],
			DescriptionEs: almanax.Bonus["es"],
			DescriptionDe: almanax.Bonus["de"],
			DescriptionPt: almanax.Bonus["pt"],
		})
		if err != nil {
			return -1, err
		}
	}

	query = `SELECT id FROM tribute WHERE item_ankama_id = ? AND quantity = ?`
	var tributeID int64
	err = r.Db.QueryRow(query, almanax.Offering.ItemId, almanax.Offering.Quantity).Scan(&tributeID)
	if err == sql.ErrNoRows {
		var itemApiUri string
		game := "dofus3"
		version := "v1"
		switch almanax.Offering.ItemCategoryId {
		case 1: // consumables
			itemApiUri = fmt.Sprintf("%s/%s/${lang}/items/consumables/%d", game, version, almanax.Offering.ItemId)
		case 2: // resources
			itemApiUri = fmt.Sprintf("%s/%s/${lang}/items/resources/%d", game, version, almanax.Offering.ItemId)
		case 0: // equipment
			itemApiUri = fmt.Sprintf("%s/%s/${lang}/items/equipment/%d", game, version, almanax.Offering.ItemId)
		case 3: // quest
			itemApiUri = fmt.Sprintf("%s/%s/${lang}/items/quest/%d", game, version, almanax.Offering.ItemId)
		case 5: // cosmetics
			itemApiUri = fmt.Sprintf("%s/%s/${lang}/items/cosmetics/%d", game, version, almanax.Offering.ItemId)
		}

		tributeID, err = r.CreateTribute(&Tribute{
			ItemNameEn:     almanax.Offering.ItemName["en"],
			ItemNameFr:     almanax.Offering.ItemName["fr"],
			ItemNameEs:     almanax.Offering.ItemName["es"],
			ItemNameDe:     almanax.Offering.ItemName["de"],
			ItemNamePt:     almanax.Offering.ItemName["pt"],
			ItemAnkamaID:   int64(almanax.Offering.ItemId),
			ItemCategoryId: almanax.Offering.ItemCategoryId,
			ItemDoduapiUri: itemApiUri,
			Quantity:       almanax.Offering.Quantity,
		})
		if err != nil {
			return -1, err
		}
	}

	query = `SELECT id FROM almanax WHERE date = ?`
	var id int64
	err = r.Db.QueryRow(query, date).Scan(&id)
	if err == sql.ErrNoRows {
		query = `
			INSERT INTO almanax (bonus_id, tribute_id, date, reward_kamas, created_at, updated_at)
			VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))`
		result, err := r.Db.Exec(query, bonusID, tributeID, date, almanax.RewardKamas)
		if err != nil {
			return 0, err
		}
		return result.LastInsertId()
	} else {
		query = `SELECT bonus_id, tribute_id, reward_kamas FROM almanax WHERE id = ?`
		var exBonusID, exTributeID, exRewardKamas int64
		err = r.Db.QueryRow(query, id).Scan(&exBonusID, &exTributeID, &exRewardKamas)
		if err != nil {
			return 0, err
		}

		if exBonusID != bonusID || exTributeID != tributeID || exRewardKamas != int64(almanax.RewardKamas) {
			err = r.UpdateAlmanax(&Almanax{
				ID:          id,
				BonusID:     bonusID,
				TributeID:   tributeID,
				Date:        date,
				RewardKamas: int64(almanax.RewardKamas),
			})
			if err != nil {
				return 0, err
			}
		}
	}

	return id, nil
}

func (r *Repository) UpdateAlmanax(almanax *Almanax) error {
	query := `
		UPDATE almanax
		SET bonus_id = ?, tribute_id = ?, date = ?, reward_kamas = ?, updated_at = datetime('now')
		WHERE id = ?`
	_, err := r.Db.Exec(query, almanax.BonusID, almanax.TributeID, almanax.Date, almanax.RewardKamas, almanax.ID)
	return err
}

func (r *Repository) UpdateFuture(data map[string]dodumap.MappedMultilangNPCAlmanaxUnity) error {
	for date, almanax := range data {
		_, err := r.CreateOrUpdate(date, &almanax)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) CreateBonusType(bonusType *BonusType) (int64, error) {
	query := `INSERT INTO bonus_types (name_id, name_en, name_fr, name_es, name_de, name_pt, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`
	result, err := r.Db.Exec(query, bonusType.NameID, bonusType.NameEn, bonusType.NameFr, bonusType.NameEs,
		bonusType.NameDe, bonusType.NamePt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func enNameToId(enName string) string {
	return strings.ToLower(strings.ReplaceAll(enName, " ", "_"))
}

func (r *Repository) GetBonusTypes() ([]BonusType, error) {
	query := `SELECT id, name_id, name_en, name_fr, name_es, name_de, name_pt
	          FROM bonus_types`
	rows, err := r.Db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]BonusType, 0)
	for rows.Next() {
		var bonusType BonusType
		err := rows.Scan(&bonusType.ID, &bonusType.NameID, &bonusType.NameEn, &bonusType.NameFr, &bonusType.NameEs,
			&bonusType.NameDe, &bonusType.NamePt)
		if err != nil {
			return nil, err
		}
		result = append(result, bonusType)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
