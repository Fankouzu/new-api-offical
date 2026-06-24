package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupReferralTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	oldDB := DB
	oldLogDB := LOG_DB
	oldQuotaForNewUser := common.QuotaForNewUser
	oldQuotaForInviter := common.QuotaForInviter
	oldQuotaForInvitee := common.QuotaForInvitee
	oldRedisEnabled := common.RedisEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled
	oldUsingSQLite := common.UsingSQLite
	oldUsingMySQL := common.UsingMySQL
	oldUsingPostgreSQL := common.UsingPostgreSQL

	common.QuotaForNewUser = 100
	common.QuotaForInviter = 30
	common.QuotaForInvitee = 20
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Log{}))

	DB = db
	LOG_DB = db

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		DB = oldDB
		LOG_DB = oldLogDB
		common.QuotaForNewUser = oldQuotaForNewUser
		common.QuotaForInviter = oldQuotaForInviter
		common.QuotaForInvitee = oldQuotaForInvitee
		common.RedisEnabled = oldRedisEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
		common.UsingSQLite = oldUsingSQLite
		common.UsingMySQL = oldUsingMySQL
		common.UsingPostgreSQL = oldUsingPostgreSQL
	})

	return db
}

func TestInsertWithInviterRecordsRelationshipAndRewards(t *testing.T) {
	db := setupReferralTestDB(t)

	inviter := User{
		Username:    "inviter",
		DisplayName: "Inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "ABCD",
		Quota:       500,
	}
	require.NoError(t, db.Create(&inviter).Error)

	invitee := User{
		Username:    "invitee",
		Password:    "password123",
		DisplayName: "Invitee",
		InviterId:   inviter.Id,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, invitee.Insert(inviter.Id))

	var savedInvitee User
	require.NoError(t, db.First(&savedInvitee, "username = ?", "invitee").Error)
	require.Equal(t, inviter.Id, savedInvitee.InviterId)
	require.Equal(t, common.QuotaForNewUser+common.QuotaForInvitee, savedInvitee.Quota)

	var savedInviter User
	require.NoError(t, db.First(&savedInviter, inviter.Id).Error)
	require.Equal(t, 1, savedInviter.AffCount)
	require.Equal(t, common.QuotaForInviter, savedInviter.AffQuota)
	require.Equal(t, common.QuotaForInviter, savedInviter.AffHistoryQuota)

	var inviteeLogCount int64
	require.NoError(t, db.Model(&Log{}).
		Where("user_id = ? AND content LIKE ?", savedInvitee.Id, "%使用邀请码赠送%").
		Count(&inviteeLogCount).Error)
	require.Equal(t, int64(1), inviteeLogCount)

	var inviterLogCount int64
	require.NoError(t, db.Model(&Log{}).
		Where("user_id = ? AND content LIKE ?", inviter.Id, "%邀请用户赠送%").
		Count(&inviterLogCount).Error)
	require.Equal(t, int64(1), inviterLogCount)
}

func TestResolveInviterByAffCodeRejectsInvalidCode(t *testing.T) {
	setupReferralTestDB(t)

	inviterId, err := ResolveInviterByAffCode("missing-code")

	require.Error(t, err)
	require.Equal(t, 0, inviterId)
}
