package model

import (
	"fmt"
	"strings"

	"github.com/GenesisKernel/go-genesis/packages/converter"
)

const (
	notificationTableSuffix = "_notifications"

	NotificationTypeSingle = 1
	NotificationTypeRole   = 2
)

// Notification structure
type Notification struct {
	ecosystem           int64
	ID                  int64  `gorm:"primary_key;not null"`
	Recipient           string `gorm:"type:jsonb(PostgreSQL)`
	Sender              string `gorm:"type:jsonb(PostgreSQL)`
	Notification        string `gorm:"type:jsonb(PostgreSQL)`
	PageParams          string `gorm:"type:jsonb(PostgreSQL)`
	ProcessingInfo      string `gorm:"type:jsonb(PostgreSQL)`
	PageName            string `gorm:"size:255"`
	DateCreated         int64
	DateStartProcessing int64
	DateClosed          int64
	Closed              bool
}

// SetTablePrefix set table Prefix
func (n *Notification) SetTablePrefix(tablePrefix string) {
	n.ecosystem = converter.StrToInt64(tablePrefix)
}

// TableName returns table name
func (n *Notification) TableName() string {
	if n.ecosystem == 0 {
		n.ecosystem = 1
	}
	return `1_notifications`
}

// GetNotificationsCount returns all unclosed notifications by users and ecosystem through role_id
// if userIDs is nil or empty then filter will be skipped
func GetNotificationsCount(ecosystemID int64, userIDs []int64) ([]map[string]string, error) {

	result := make([]map[string]string, 0, 16)
	for _, userID := range userIDs {
		roles, err := GetMemberRoles(nil, ecosystemID, userID)
		if err != nil {
			return nil, err
		}
		roleList := make([]string, 0, len(roles))
		for _, role := range roles {
			roleList = append(roleList, converter.Int64ToStr(role))
		}
		query := fmt.Sprintf(`SELECT '%d' as "recipient_id", recipient->>'role_id' as "role_id", count(*) cnt	FROM "1_notifications" 
		 WHERE ecosystem='%d' AND closed = 0 AND ((notification->>'type' = '1' and recipient->>'member_id' = '%[1]d' ) or
		   (notification->>'type' = '2' and (recipient->>'role_id' IN ('%[4]s') and 
		   ( date_start_processing is null or processing_info->>'member_id' = '%[1]d'))))
		GROUP BY 1,2`, userID, ecosystemID, strings.Join(roleList, "','"))
		list, err := GetAllTransaction(nil, query, -1)
		if err != nil {
			return nil, err
		}
		result = append(result, list...)
	}
	return result, nil
}

func getNotificationCountFilter(users []int64, ecosystemID int64) (filter string, params []interface{}) {
	filter = fmt.Sprintf(` WHERE closed = 0 and ecosystem = '%d' `, ecosystemID)

	if len(users) > 0 {
		filter += `AND recipient->>'member_id' IN (?) `
		usersStrs := []string{}
		for _, user := range users {
			usersStrs = append(usersStrs, converter.Int64ToStr(user))
		}
		params = append(params, usersStrs)
	}

	return
}
