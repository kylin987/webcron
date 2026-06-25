package libs

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
)

var emailPattern = regexp.MustCompile("[\\w!#$%&'*+/=?^_`{|}~-]+(?:\\.[\\w!#$%&'*+/=?^_`{|}~-]+)*@(?:[\\w](?:[\\w-]*[\\w])?\\.)+[a-zA-Z0-9](?:[\\w-]*[\\w])?")

func Md5(buf []byte) string {
	hash := md5.New()
	hash.Write(buf)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func SizeFormat(size float64) string {
	units := []string{"Byte", "KB", "MB", "GB", "TB"}
	n := 0
	for size > 1024 {
		size /= 1024
		n += 1
	}

	return fmt.Sprintf("%.2f %s", size, units[n])
}

func IsEmail(b []byte) bool {
	return emailPattern.Match(b)
}

// 强密码字符集：大小写字母 + 数字 + 常见符号，避免易混淆字符 0/O/l/I/1
var passwordLetters = []byte("ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789!@#$%^&*()-_=+")

// GeneratePassword 生成指定长度的随机强密码，包含大小写字母、数字与符号。
func GeneratePassword(length int) string {
	if length < 8 {
		length = 16
	}
	buf := make([]byte, length)
	for i := range buf {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(passwordLetters))))
		buf[i] = passwordLetters[n.Int64()]
	}
	return string(buf)
}
