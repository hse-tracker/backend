package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hse-tracker/backend/internal/storage"
	"github.com/jmoiron/sqlx"
)

func ValidateInitData(initData, token string) (int64, error) {
	vals, err := url.ParseQuery(initData)
	if err != nil {
		return 0, err
	}
	hash := vals.Get("hash")
	vals.Del("hash")

	var keys []string
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var dataCheckArr []string
	for _, k := range keys {
		dataCheckArr = append(dataCheckArr, fmt.Sprintf("%s=%s", k, vals.Get(k)))
	}
	dataCheckString := strings.Join(dataCheckArr, "\n")

	secretKey := hmac.New(sha256.New, []byte("WebAppData"))
	secretKey.Write([]byte(token))
	secret := secretKey.Sum(nil)

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(dataCheckString))
	if hex.EncodeToString(h.Sum(nil)) != hash {
		return 0, fmt.Errorf("invalid hash")
	}

	var user struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(vals.Get("user")), &user); err != nil {
		return 0, err
	}
	return user.ID, nil
}

type TokenClaims struct {
	UserID  int64 `json:"user_id"`
	GroupID int64 `json:"group_id"`
	jwt.RegisteredClaims
}

// user register request data type
type RegisterRequest struct {
	InitData string `json:"init_data"`
	FullName string `json:"full_name"`
	Group    string `json:"group"`
}

// user register response data type
type RegisterResponse struct {
	Token string `json:"token"`
}

// register handler
func RegisterHandler(db *sqlx.DB, secret string, tgToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decoding JSON
		var req RegisterRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		log.Printf("[API] Register request: FullName='%s', Group='%s'", req.FullName, req.Group)

		var tgID int64

		// ВРЕМЕННЫЙ БАЙПАС ДЛЯ ТЕСТОВ НА localhost
		// if req.InitData == "mock_local_data" {
		// 	tgID = 123456789
		// 	fmt.Println("⚠️ ИСПОЛЬЗОВАН ТЕСТОВЫЙ БАЙПАС АВТОРИЗАЦИИ ⚠️")
		// } else {

		// }

		tgID, err = ValidateInitData(req.InitData, tgToken)
		if err != nil {
			http.Error(w, "unauthorized telegram data", http.StatusUnauthorized)
			return
		}

		user, err := storage.RegisterUser(db, tgID, req.FullName, req.Group)
		if err != nil {
			fmt.Printf("db error while registering user: %s\n", err)
			http.Error(w, "internal error", 500)
			return
		}
		fmt.Printf("user created: \n%s, \n%d\n", user.FullName, user.GroupID)

		claims := TokenClaims{
			UserID:  user.ID,
			GroupID: user.GroupID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(secret))
		if err != nil {
			fmt.Printf("jwt error: %s\n", err)
			http.Error(w, "internal error", 500)
			return
		}

		// sending response
		resp := RegisterResponse{
			Token: signedToken,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return
		}
	}
}
