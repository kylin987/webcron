package models

import (
	"github.com/astaxie/beego/orm"
	"time"
)

const (
	OP_RESULT_OK  = 0
	OP_RESULT_ERR = -1
)

type OperationLog struct {
	Id            int64
	UserId        int
	UserName      string
	Action        string
	TargetType    string
	TargetId      int
	Result        int
	Ip            string
	UserAgent     string
	RequestMethod string
	RequestPath   string
	Detail        string
	CreateTime    int64
}

func (o *OperationLog) TableName() string {
	return TableName("operation_log")
}

func OperationLogAdd(log *OperationLog) (int64, error) {
	if log.CreateTime == 0 {
		log.CreateTime = time.Now().Unix()
	}
	return orm.NewOrm().Insert(log)
}

func OperationLogGetById(id int64) (*OperationLog, error) {
	log := &OperationLog{
		Id: id,
	}
	err := orm.NewOrm().Read(log)
	if err != nil {
		return nil, err
	}
	return log, nil
}

func OperationLogGetList(page, pageSize int, filters ...interface{}) ([]*OperationLog, int64) {
	offset := (page - 1) * pageSize
	list := make([]*OperationLog, 0)

	query := orm.NewOrm().QueryTable(TableName("operation_log"))
	if len(filters) > 0 {
		l := len(filters)
		for k := 0; k < l; k += 2 {
			query = query.Filter(filters[k].(string), filters[k+1])
		}
	}
	total, _ := query.Count()
	query.OrderBy("-id").Limit(pageSize, offset).All(&list)

	return list, total
}
