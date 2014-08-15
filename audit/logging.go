package audit

import "github.com/sprucehealth/backend/libs/golog"

func LogAction(accountID int64, component, action string, additional map[string]interface{}) {
	ctx := make([]interface{}, 0, 2*(3+len(additional)))
	ctx = append(ctx,
		"account_id", accountID,
		"component", component,
		"action", action,
	)
	for k, v := range additional {
		ctx = append(ctx, k, v)
	}
	golog.Context(ctx...).Infof("audit")
}
