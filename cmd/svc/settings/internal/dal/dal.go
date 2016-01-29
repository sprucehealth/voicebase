package dal

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/sprucehealth/backend/cmd/svc/settings/internal/models"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/ptr"
)

const (
	nodeIDColumn    = "nodeID"
	configKeyColumn = "key"
	dataColumn      = "data"
)

type DAL interface {
	GetConfigs(keys []string) ([]*models.Config, error)
	SetConfigs(config []*models.Config) error
	GetValues(nodeID string, keys []*models.ConfigKey) ([]*models.Value, error)
	SetValues(nodeID string, values []*models.Value) error
}

type dal struct {
	db                      dynamodbiface.DynamoDBAPI
	tableNameSettings       string
	tableNameSettingsConfig string
}

func New(db dynamodbiface.DynamoDBAPI, tableNameSettings, tableNameSettingConfigs string) DAL {
	return &dal{
		db:                      db,
		tableNameSettings:       tableNameSettings,
		tableNameSettingsConfig: tableNameSettingConfigs,
	}
}

func (d *dal) GetConfigs(keys []string) ([]*models.Config, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	dbKeys := make([]map[string]*dynamodb.AttributeValue, len(keys))
	for i, key := range keys {
		dbKeys[i] = map[string]*dynamodb.AttributeValue{
			configKeyColumn: {S: ptr.String(key)},
		}
	}

	res, err := d.db.BatchGetItem(&dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			d.tableNameSettingsConfig: {
				AttributesToGet: []*string{ptr.String(dataColumn)},
				ConsistentRead:  ptr.Bool(true),
				Keys:            dbKeys,
			},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	configs := make([]*models.Config, len(keys))
	for i, r := range res.Responses[d.tableNameSettingsConfig] {
		var c models.Config
		if err := c.Unmarshal(r[dataColumn].B); err != nil {
			return nil, errors.Trace(err)
		}
		configs[i] = &c
	}
	return configs, nil
}

func (d *dal) SetConfigs(configs []*models.Config) error {

	if len(configs) == 0 {
		return nil
	}

	req := make([]*dynamodb.WriteRequest, len(configs))
	for i, c := range configs {
		b, err := c.Marshal()
		if err != nil {
			return errors.Trace(err)
		}

		req[i] = &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					configKeyColumn: {S: ptr.String(c.Key)},
					dataColumn:      {B: b},
				},
			},
		}
	}

	_, err := d.db.BatchWriteItem(&dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			d.tableNameSettingsConfig: req,
		},
	})

	return errors.Trace(err)
}

func (d *dal) GetValues(nodeID string, keys []*models.ConfigKey) ([]*models.Value, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	dbKeys := make([]map[string]*dynamodb.AttributeValue, len(keys))
	for i, key := range keys {
		dbKeys[i] = map[string]*dynamodb.AttributeValue{
			nodeIDColumn:    {S: ptr.String(nodeID)},
			configKeyColumn: {S: ptr.String(models.FlattenConfigKey(key))},
		}
	}

	res, err := d.db.BatchGetItem(&dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			d.tableNameSettings: {
				AttributesToGet: []*string{ptr.String(dataColumn)},
				ConsistentRead:  ptr.Bool(true),
				Keys:            dbKeys,
			},
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	values := make([]*models.Value, 0, len(keys))
	for _, r := range res.Responses[d.tableNameSettings] {
		var v models.Value
		if err := v.Unmarshal(r[dataColumn].B); err != nil {
			return nil, errors.Trace(err)
		}
		values = append(values, &v)
	}

	return values, nil
}

func (d *dal) SetValues(nodeID string, values []*models.Value) error {
	if len(values) == 0 {
		return nil
	}

	req := make([]*dynamodb.WriteRequest, len(values))
	for i, v := range values {
		b, err := v.Marshal()
		if err != nil {
			return errors.Trace(err)
		}

		req[i] = &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: map[string]*dynamodb.AttributeValue{
					nodeIDColumn:    {S: ptr.String(nodeID)},
					configKeyColumn: {S: ptr.String(models.FlattenConfigKey(v.Key))},
					dataColumn:      {B: b},
				},
			},
		}
	}

	_, err := d.db.BatchWriteItem(&dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			d.tableNameSettings: req,
		},
	})

	return errors.Trace(err)
}
