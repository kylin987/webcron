package models

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"time"
)

type User struct {
	Id             int
	UserName       string
	Password       string
	Salt           string
	MfaSecret      string
	MfaEnabled     int
	MfaConfirmedAt int64
	Email          string
	LastLogin      int64
	LastIp         string
	Status         int
	CreateTime     int64
}

func (u *User) TableName() string {
	return TableName("user")
}

func (u *User) Update(fields ...string) error {
	if _, err := orm.NewOrm().Update(u, fields...); err != nil {
		return err
	}
	return nil
}

func UserAdd(user *User) (int64, error) {
	if user.UserName == "" {
		return 0, fmt.Errorf("用户名不能为空")
	}
	if user.Password == "" {
		return 0, fmt.Errorf("密码不能为空")
	}
	if user.CreateTime == 0 {
		user.CreateTime = time.Now().Unix()
	}
	return orm.NewOrm().Insert(user)
}

func UserGetById(id int) (*User, error) {
	u := new(User)

	err := orm.NewOrm().QueryTable(TableName("user")).Filter("id", id).One(u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func UserGetByName(userName string) (*User, error) {
	u := new(User)

	err := orm.NewOrm().QueryTable(TableName("user")).Filter("user_name", userName).One(u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func UserUpdate(user *User, fields ...string) error {
	_, err := orm.NewOrm().Update(user, fields...)
	return err
}

func UserGetList(page, pageSize int, filters ...interface{}) ([]*User, int64) {
	offset := (page - 1) * pageSize
	list := make([]*User, 0)

	query := orm.NewOrm().QueryTable(TableName("user"))
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
