package datastore

import (
	"context"
	"strings"
	"time"

	"millionaire/internal/models"

	"github.com/uptrace/bun"
)

func CreateTableUser(ctx context.Context, db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*models.User)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.User)(nil)).Index("index_user_cast(id_as_text)").IfNotExists().ColumnExpr("cast(id as text)").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.User)(nil)).Index("index_user_ref_code").IfNotExists().Column("ref_code").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.User)(nil)).Index("index_user_inviter_id").IfNotExists().Column("inviter_id").Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewRaw(`
		alter table "user"
			add if not exists lifeline_balance int default 0;
		alter table "user"
    		alter column created_at set default current_timestamp;
		alter table "user"
			add if not exists chat_status varchar default null;
		alter table "user"
    		add if not exists avatar varchar;`).Exec(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewCreateIndex().Model((*models.User)(nil)).Index("index_user_chat_status").IfNotExists().Column("chat_status").Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func FindUserByID(ctx context.Context, db *bun.DB, userID string) (*models.User, error) {
	var user models.User
	err := db.NewSelect().Model(&user).Where("id = ?", userID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func CheckUserExists(ctx context.Context, db *bun.DB, userID string) (bool, error) {
	var user models.User
	err := db.NewSelect().Model(&user).Where("id = ?", userID).Scan(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

func CreateUser(ctx context.Context, db *bun.DB, user *models.User) (*models.User, error) {
	_, err := db.NewInsert().Model(user).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// if the user is not found, return nil
func FindUserByUsername(ctx context.Context, db *bun.DB, username string) (*models.User, error) {
	var user models.User
	err := db.NewSelect().Model(&user).Where("username = ?", strings.ToLower(username)).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func EditUser(ctx context.Context, db *bun.DB, user *models.User) (*models.User, error) {
	_, err := db.NewUpdate().Model(user).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func GetUserByCustomRefCode(ctx context.Context, db *bun.DB, customRefCode string) (*models.User, error) {
	var user models.User
	err := db.NewSelect().Model(&user).
		Where("ref_code = ?", customRefCode).
		WhereOr("cast(id as text) = ?", customRefCode).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func CheckRefCodeExists(ctx context.Context, db *bun.DB, customRefCode string) (bool, error) {
	var user models.User
	err := db.NewSelect().Model(&user).Where("ref_code = ?", customRefCode).Scan(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func GetUsersByInviter(ctx context.Context, db *bun.DB, inviterID int64) ([]*models.User, error) {
	var users []*models.User
	err := db.NewSelect().Model(&users).Where("inviter_id = ?", inviterID).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func CountInviteesByUserId(ctx context.Context, db *bun.DB, userID string) (int, error) {
	count, err := db.NewSelect().Model((*models.User)(nil)).Where("inviter_id = ?", userID).Count(ctx)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func CountUsers(ctx context.Context, db *bun.DB) (int, error) {
	count, err := db.NewSelect().Model((*models.User)(nil)).Count(ctx)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func GetUsersSortedByCreatedAt(ctx context.Context, db *bun.DB, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	err := db.NewSelect().Model(&users).Order("created_at ASC").Limit(limit).Offset(offset).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func GetChatAvailabledUsersSortedByCreatedAt(ctx context.Context, db *bun.DB, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	err := db.NewSelect().Model(&users).Where("chat_status is null").Order("created_at ASC").Limit(limit).Offset(offset).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func GetTopInvitedUsers(ctx context.Context, db *bun.DB, withRefCode bool, limit int) ([]*models.User, error) {
	var users []*models.User
	var err error
	if withRefCode {
		err = db.NewSelect().Model(&users).Where("ref_code IS NOT NULL").Order("total_invites DESC").Limit(limit).Scan(ctx)
	} else {
		err = db.NewSelect().Model(&users).Order("total_invites DESC").Limit(limit).Scan(ctx)
	}

	if err != nil {
		return nil, err
	}

	return users, nil
}

func CountTopInvitedUsers(ctx context.Context, db *bun.DB, limit int) ([]*models.Ref, error) {
	var refs []*models.Ref
	err := db.NewSelect().
		ColumnExpr("u.username, invite_count.count, u.id").
		TableExpr("\"user\" u").
		Join("LEFT JOIN (SELECT inviter_id, COUNT(*) count FROM \"user\" GROUP BY inviter_id) as invite_count ON u.id = invite_count.inviter_id").
		Order("invite_count.count DESC NULLS LAST").
		Limit(limit).
		Scan(ctx, &refs)
	if err != nil {
		return nil, err
	}

	return refs, nil
}

func GetCountdownCompletedUsers(ctx context.Context, db *bun.DB, gameId string, now time.Time) ([]*models.User, error) {
	var users []*models.User
	err := db.NewSelect().Model(&users).Where("countdown <= ?", now).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func UpdateUserProfile(ctx context.Context, db *bun.DB, user *models.User) (*models.User, error) {
	_, err := db.NewUpdate().Model(user).
		Set("username = ?", user.Username).
		Set("first_name = ?", user.FirstName).
		Set("last_name = ?", user.LastName).
		Set("photo_url = ?", user.PhotoURL).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func UpdateUserBonusMilestone(ctx context.Context, db *bun.DB, user *models.UserGame) (*models.UserGame, error) {
	_, err := db.NewUpdate().Model(user).
		Set("current_bonus_milestone = ?", user.CurrentBonusMilestone).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func GetUsersByLimit(ctx context.Context, db *bun.DB, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	err := db.NewSelect().Model(&users).Limit(limit).Offset(offset).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func AddInviteRef(ctx context.Context, db *bun.DB, inviteeID string, inviterID string) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model((*models.User)(nil)).
			Set("inviter_id = ?", inviterID).
			Where("id=?", inviteeID).
			Where("inviter_id is null").Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewUpdate().
			Model((*models.User)(nil)).
			Set("total_invites = total_invites + 1").
			Where("id = ?", inviterID).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}

func ChangeUserLifelineBalance(ctx context.Context, db *bun.DB, userID string, number int) error {
	_, err := db.NewUpdate().
		Model((*models.User)(nil)).
		Set("lifeline_balance = lifeline_balance + ?", number).
		Where("id = ?", userID).
		Exec(ctx)
	return err
}

func GetUserFriendList(ctx context.Context, db *bun.DB, userID string) ([]*models.Friend, error) {
	var friends []*models.Friend
	err := db.NewSelect().
		Model(&friends).
		ColumnExpr("ur.id, ur.first_name, ur.last_name, ur.username, us.gems").TableExpr("user u").
		Join("LEFT JOIN user ur ON u.id = ur.inviter_id").
		Join("LEFT JOIN (SELECT u.id, SUM(ug.gems) gems FROM user u LEFT JOIN user_gem ug ON u.id = ug.user_id GROUP BY u.id) us ON ur.id = us.id").
		Where("u.id = ?", userID).
		Order("us.gems DESC NULLS LAST").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return friends, nil
}

func GetUserFriendListPaging(ctx context.Context, db *bun.DB, userID string, limit, offset int) ([]*models.Friend, error) {
	var friends []*models.Friend

	err := db.NewSelect().
		ColumnExpr("ur.id, ur.first_name, ur.last_name, ur.username, us.gems, (us.gems IS NOT NULL AND us.gems > 15) as validated, ub.id is not null as claimed").TableExpr("\"user\" ur").
		Join("LEFT JOIN (SELECT u.id, SUM(ug.gems) gems FROM \"user\" u LEFT JOIN user_gem ug ON u.id = ug.user_id WHERE ug.user_id IN (SELECT u.id FROM \"user\" u WHERE u.inviter_id=?) GROUP BY u.id) us ON ur.id = us.id", userID).
		Join("LEFT JOIN user_boost ub ON ur.inviter_id = ub.user_id AND ur.id::varchar = ub.source").
		Where("ur.inviter_id = '?'", userID).
		Where("us.gems >= 5").
		Order("validated desc").
		Order("claimed").
		Order("us.gems DESC NULLS LAST").
		Limit(limit).
		Offset(offset).
		Scan(ctx, &friends)
	if err != nil {
		return nil, err
	}

	return friends, nil
}

func GetOnlyClaimableFriends(ctx context.Context, db *bun.DB, userID string) ([]*models.Friend, error) {
	var friends []*models.Friend

	err := db.NewSelect().
		ColumnExpr("ur.id, ur.first_name, ur.last_name, ur.username, us.gems").TableExpr("\"user\" ur").
		Join("LEFT JOIN (SELECT u.id, SUM(ug.gems) gems FROM \"user\" u LEFT JOIN user_gem ug ON u.id = ug.user_id WHERE ug.user_id IN (SELECT u.id FROM \"user\" u WHERE u.inviter_id=?) GROUP BY u.id) us ON ur.id = us.id", userID).
		Join("LEFT JOIN user_boost ub ON ur.inviter_id = ub.user_id AND ur.id::varchar = ub.source").
		Where("ur.inviter_id = '?'", userID).
		Where("us.gems > 15").
		Where("ub.id is null").
		Scan(ctx, &friends)
	if err != nil {
		return nil, err
	}

	return friends, nil
}

func CountFriends(ctx context.Context, db *bun.DB, userID string) (int, error) {
	count, err := db.NewSelect().
		ColumnExpr("ur.id, ur.first_name, ur.last_name, ur.username, us.gems, (us.gems IS NOT NULL AND us.gems > 15) as validated, ub.id is not null as claimed").TableExpr("\"user\" ur").
		Join("LEFT JOIN (SELECT u.id, SUM(ug.gems) gems FROM \"user\" u LEFT JOIN user_gem ug ON u.id = ug.user_id WHERE ug.user_id IN (SELECT u.id FROM \"user\" u WHERE u.inviter_id=?) GROUP BY u.id) us ON ur.id = us.id", userID).
		Join("LEFT JOIN user_boost ub ON ur.inviter_id = ub.user_id AND ur.id::varchar = ub.source").
		Where("ur.inviter_id = '?'", userID).
		Where("us.gems >= 5").
		Count(ctx)

	if err != nil {
		return 0, err
	}

	return count, nil
}

func UpdateUserInvitees(ctx context.Context, db *bun.DB, userID string, totalInvites int) error {
	_, err := db.NewUpdate().
		Model((*models.User)(nil)).
		Set("total_invites = ?", totalInvites).
		Where("id = ?", userID).
		Exec(ctx)
	return err
}

func UpdateUserStatus(ctx context.Context, db *bun.DB, userID string, status string) error {
	_, err := db.NewUpdate().
		Model((*models.User)(nil)).
		Set("chat_status = ?", status).
		Where("id = ?", userID).
		Exec(ctx)
	return err
}
