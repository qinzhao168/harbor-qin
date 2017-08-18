// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils"

	"github.com/vmware/harbor/src/common/utils/log"
)

// GetUser ...
func GetUser(query models.User) (*models.User, error) {

	o := GetOrmer()

	sql := `select user_id, user_name, email, displayname, avatar, confirm_code, api_token,
		admin_role, creation_time, last_login_time
		from tenx_users u
		where migrated = 0 `
	queryParam := make([]interface{}, 1)
	if query.UserID != 0 {
		sql += ` and user_id = ? `
		queryParam = append(queryParam, query.UserID)
	}

	if query.Username != "" {
		sql += ` and user_name = ? `
		queryParam = append(queryParam, query.Username)
	}

	if query.ResetUUID != "" {
		sql += ` and confirm_code = ? `
		queryParam = append(queryParam, query.ResetUUID)
	}

	var u []models.User
	n, err := o.Raw(sql, queryParam).QueryRows(&u)

	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}

	if n > 1 {
		return nil, fmt.Errorf("got more than one user when executing: %s param: %v", sql, queryParam)
	}

	return &u[0], nil
}

// LoginByDb is used for user to login with database auth mode.
// TenxCloud: Authenticate using user and token
func LoginByDb(auth models.AuthModel) (*models.User, error) {
	o := GetOrmer()
	var users []models.User
	n, err := o.Raw(`select * from tenx_users where (user_name = ? or email = ?) and migrated = 0`,
		auth.Principal, auth.Principal).QueryRows(&users)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}

	user := users[0]

	if user.Password != encodeMD5(auth.Password) {
		return nil, nil
	}

	user.Password = "" //do not return the password

	return &user, nil
}

// EncodeMD5 TenxCloud Added
func encodeMD5(str string) string {
	h := md5.New()
	io.WriteString(h, str)

	strSum := h.Sum(nil)
	res := hex.EncodeToString(strSum)
	return res
}

// ListUsers lists all users according to different conditions.
func ListUsers(query models.User) ([]models.User, error) {
	o := GetOrmer()
	u := []models.User{}
	sql := `select  user_id, user_name, email, displayname, avatar, confirm_code, api_token,
		admin_role, creation_time, last_login_time
		from tenx_users u
		where u.migrated = 0 and u.user_id != 1 `

	queryParam := make([]interface{}, 1)
	if query.Username != "" {
		sql += ` and user_name like ? `
		queryParam = append(queryParam, "%"+escape(query.Username)+"%")
	}
	sql += ` order by user_id desc `

	_, err := o.Raw(sql, queryParam).QueryRows(&u)
	return u, err
}

// ToggleUserAdminRole gives a user admin role.
func ToggleUserAdminRole(userID, hasAdmin int) error {
	o := GetOrmer()
	queryParams := make([]interface{}, 1)
	sql := `update tenx_users set admin_role = ? where user_id = ?`
	queryParams = append(queryParams, hasAdmin)
	queryParams = append(queryParams, userID)
	r, err := o.Raw(sql, queryParams).Exec()
	if err != nil {
		return err
	}

	if _, err := r.RowsAffected(); err != nil {
		return err
	}

	return nil
}

// ChangeUserPassword ...
func ChangeUserPassword(u models.User, oldPassword ...string) (err error) {
	if len(oldPassword) > 1 {
		return errors.New("wrong numbers of params")
	}

	o := GetOrmer()

	var r sql.Result
	salt := utils.GenerateRandomString()
	if len(oldPassword) == 0 {
		//In some cases, it may no need to check old password, just as Linux change password policies.
		r, err = o.Raw(`update tenx_users set password=?, api_token=? where user_id=?`, utils.Encrypt(u.Password, salt), salt, u.UserID).Exec()
	} else {
		r, err = o.Raw(`update tenx_users set password=?, api_token=? where user_id=? and password = ?`, utils.Encrypt(u.Password, salt), salt, u.UserID, utils.Encrypt(oldPassword[0], u.Salt)).Exec()
	}

	if err != nil {
		return err
	}
	c, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if c == 0 {
		return errors.New("no record has been modified, change password failed")
	}

	return nil
}

// ResetUserPassword ...
func ResetUserPassword(u models.User) error {
	o := GetOrmer()
	r, err := o.Raw(`update tenx_users set password=?, confirm_code=? where confirm_code=?`, utils.Encrypt(u.Password, u.Salt), "", u.ResetUUID).Exec()
	if err != nil {
		return err
	}
	count, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("no record be changed, reset password failed")
	}
	return nil
}

// UpdateUserResetUUID ...
func UpdateUserResetUUID(u models.User) error {
	o := GetOrmer()
	_, err := o.Raw(`update tenx_users set confirm_code=? where email=?`, u.ResetUUID, u.Email).Exec()
	return err
}

// CheckUserPassword checks whether the password is correct.
func CheckUserPassword(query models.User) (*models.User, error) {

	currentUser, err := GetUser(query)
	if err != nil {
		return nil, err
	}
	if currentUser == nil {
		return nil, nil
	}

	sql := `select user_id, user_name, api_token from tenx_users where migrated = 0 and user_name = ? and password = ?`
	queryParam := make([]interface{}, 1)
	queryParam = append(queryParam, currentUser.Username)
	queryParam = append(queryParam, utils.Encrypt(query.Password, currentUser.Salt))
	o := GetOrmer()
	var user []models.User

	n, err := o.Raw(sql, queryParam).QueryRows(&user)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		log.Warning("User principal does not match password. Current:", currentUser)
		return nil, nil
	}

	return &user[0], nil
}

// DeleteUser ...
func DeleteUser(userID int) error {
	o := GetOrmer()

	user, err := GetUser(models.User{
		UserID: userID,
	})
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%s#%d", user.Username, user.UserID)
	email := fmt.Sprintf("%s#%d", user.Email, user.UserID)

	_, err = o.Raw(`update tenx_users
		set migrated = 1, user_name = ?, email = ?
		where user_id = ?`, name, email, userID).Exec()
	return err
}

// ChangeUserProfile ...
func ChangeUserProfile(user models.User) error {
	o := GetOrmer()
	if _, err := o.Update(&user, "Email", "Realname", "Comment"); err != nil {
		log.Errorf("update user failed, error: %v", err)
		return err
	}
	return nil
}
