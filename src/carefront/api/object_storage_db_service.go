package api

func (d *DataService) CreateNewUploadCloudObjectRecord(bucket, key, region string) (int64, error) {
	res, err := d.db.Exec(`insert into object_storage (bucket, storage_key, status, region_id) 
								values (?, ?, 'CREATING', (select id from region where region_tag = ?))`, bucket, key, region)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (d *DataService) UpdateCloudObjectRecordToSayCompleted(id int64) error {
	_, err := d.db.Exec("update object_storage set status='ACTIVE' where id = ?", id)
	return err
}
