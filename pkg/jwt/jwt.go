package jwt

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

const (
	tokenExpireDuration = time.Hour * 24
)

var (
	CustomSecret = []byte("小熙")
)

type UserClaims struct {
	// 可根据需要自行添加字段
	Username string `json:"username"`
	ID       uint   `json:"id"`
}

type CustomClaims struct {
	UserClaims
	jwt.RegisteredClaims // 内嵌标准的声明
}

// GenToken 生成token
func GenToken(Id uint, username string) (string, error) {
	user := UserClaims{
		Username: username,
		ID:       Id,
	}
	// 创建一个我们自己的声明
	claims := CustomClaims{
		user, // 自定义字段
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpireDuration)),
			Issuer:    "my-project", // 签发人
		},
	}
	// 使用指定的签名方法创建签名对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// 使用指定的secret签名并获得完整的编码后的字符串token
	return token.SignedString(CustomSecret)
}

// ParseToken 解析JWT
func ParseToken(tokenString string) (*CustomClaims, error) {
	// 解析token
	// 如果是自定义Claim结构体则需要使用 ParseWithClaims 方法
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (i interface{}, err error) {
		// 直接使用标准的Claim则可以直接使用Parse方法
		//token, err := jwt.Parse(tokenString, func(token *jwt.Token) (i interface{}, err error) {
		return CustomSecret, nil
	})
	if err != nil {
		return nil, err
	}
	// 对token对象中的Claim进行类型断言
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid { // 校验token
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
