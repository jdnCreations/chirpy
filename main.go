package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jdnCreations/chirpy/internal/auth"
	"github.com/jdnCreations/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type User struct {
  ID uuid.UUID `json:"id"`
  CreatedAt time.Time `json:"created_at"`
  UpdatedAt time.Time `json:"updated_at"`
  Email string `json:"email"`
  Token string `json:"token"`
  RefreshToken string `json:"refresh_token"`
  IsChirpyRed bool `json:"is_chirpy_red"`
}

type UserInfo struct {
  Password string `json:"password"`
  Email string `json:"email"`
}

type Chirp struct {
  ID uuid.UUID `json:"id"`
  CreatedAt time.Time `json:"created_at"`
  UpdatedAt time.Time `json:"updated_at"`
  Body string `json:"body"`
  UserID uuid.UUID `json:"user_id"`
}

type apiConfig struct {
	fileserverHits atomic.Int32
  db *database.Queries
  secret string
}

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})	
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	w.Header().Set("Content-Type", "text/html")
	htmlTemplate := `
	<html>
	<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
	</body>
	</html>
	`
	responseStr := fmt.Sprintf(htmlTemplate, hits)
	w.Write([]byte(responseStr))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func cleanText(text string) string {
  badwords := [3]string{"kerfuffle", "sharbert", "fornax"}

  split := strings.Split(text, " ")
  for i, word := range split {
    for _, bad := range badwords {
      if strings.ToLower(word) == bad {
        split[i] = "****"
        break 
      }
    }
  }
  return strings.Join(split, " ")
}


func respondWithError(w http.ResponseWriter, code int, msg string) {
  type returnErr struct {
    Error string `json:"error"`
  }

  respBody := returnErr{
    Error: msg,
  }
  dat, err := json.Marshal(respBody)
  if err != nil {
    log.Printf("Error marshalling JSON %s", err)
    w.WriteHeader(500)
    return
  }
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(code)
  w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
  dat, err := json.Marshal(payload)
  if err != nil {
    respondWithError(w, 500, "Error marshalling JSON")
    return
  }

  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(code)
  w.Write(dat)
}

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, r *http.Request) {
  decoder := json.NewDecoder(r.Body)
  params := UserInfo{}
  err := decoder.Decode(&params)
  if err != nil {
    respondWithError(w, 500, err.Error())
    return
  } 

  hashed, err := auth.HashPassword(params.Password)
  if err != nil {
    respondWithError(w, 500, err.Error())
    return
  }
  
  u, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
    Email: params.Email,
    HashedPassword: hashed,
  })
  if err != nil {
    respondWithError(w, 422, "Could not create user")
    return
  }
  
  user := User{
    ID: u.ID,
    CreatedAt: u.CreatedAt,
    UpdatedAt: u.UpdatedAt,
    Email: u.Email,
    IsChirpyRed: u.IsChirpyRed.Bool,
  }

  respondWithJSON(w, 201, user)
}

func (cfg *apiConfig) handlerChirp(w http.ResponseWriter, r *http.Request) {
  type parameters struct {
    Body string `json:"body"`
  }

  token, err := auth.GetBearerToken(r.Header)
  if err != nil {
    respondWithError(w, 401, "Invalid Bearer")
    return
  }

  userID, err := auth.ValidateJWT(token, cfg.secret)
  if err != nil {
    respondWithError(w, 401, "Unauthorized")
    return
  }

  decoder := json.NewDecoder(r.Body)
  params := parameters{}
  err = decoder.Decode(&params)
  if err != nil {
    respondWithError(w, 500, "Something went wrong")
    return
  }

  if (len(params.Body) > 400) {
    respondWithError(w, 400, "Chirp is too long")
    return 
  } 

  cleanedtext := cleanText(params.Body)


  chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
    Body: cleanedtext,
    UserID: uuid.NullUUID{UUID: userID, Valid:  true}, 
  })
  if err != nil {
    respondWithError(w, 422, "Could not create chirp")
    return
  }

  resData := Chirp{
    ID: chirp.ID,
    CreatedAt: chirp.CreatedAt,
    UpdatedAt: chirp.UpdatedAt,
    Body: chirp.Body,
    UserID: chirp.UserID.UUID,
  }

  respondWithJSON(w, 201, resData)
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
  chirps, err := cfg.db.GetAllChirps(r.Context())
  if err != nil {
    respondWithError(w, 500, "Could not fetch chirps")
    return
  }

  dat := []Chirp{}

  for i := range chirps {
    dat = append(dat, Chirp{ID: chirps[i].ID, CreatedAt: chirps[i].CreatedAt, UpdatedAt: chirps[i].UpdatedAt, Body: chirps[i].Body, UserID: chirps[i].UserID.UUID })
  }

  respondWithJSON(w, 200, dat)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
  id := r.PathValue("chirpID")

  chirpId, err := uuid.Parse(id)
  if err != nil {
    respondWithError(w, 500, "Invalid chirp ID")
    return
  }

  
  chirp, err := cfg.db.GetChirpById(r.Context(), chirpId)
  if err != nil {
    respondWithError(w, 404, "Chirp not found")
    return
  }

  dat := Chirp{
    ID: chirp.ID,
    CreatedAt: chirp.CreatedAt,
    UpdatedAt: chirp.UpdatedAt,
    Body: chirp.Body,
    UserID: chirp.UserID.UUID,
  }

  respondWithJSON(w, 200, dat)
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
  decoder := json.NewDecoder(r.Body)
  params := UserInfo{}
  err := decoder.Decode(&params)
  if err != nil {
    respondWithError(w, 500, err.Error())
    return
  } 

  user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
  if err != nil {
    respondWithError(w, 401, err.Error())
    return
  }

  err = auth.CheckPasswordHash(params.Password, user.HashedPassword)
  if err != nil {
    respondWithError(w, 401, err.Error())
    return
  }

  token, err := auth.MakeJWT(user.ID, cfg.secret, time.Duration(1 * time.Hour))
  if err != nil {
    respondWithError(w, 500, "could not make JWT")
    return
  }

  refreshToken, err := auth.MakeRefreshToken()
  if err != nil {
    respondWithError(w, 500, "could not make refresh token")
    return
  }

  sixtyDays := time.Duration(1440 * time.Hour)
  
  _, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
    Token: refreshToken,
    UserID: user.ID,
    ExpiresAt: time.Now().UTC().Add(sixtyDays),
  })
  if err != nil {
    respondWithError(w, 500, "could not save refresh token to database")
  }

  userDat := User{
    ID: user.ID,
    CreatedAt: user.CreatedAt,
    UpdatedAt: user.UpdatedAt,
    Email: user.Email,
    Token: token,
    RefreshToken: refreshToken,
    IsChirpyRed: user.IsChirpyRed.Bool,
  }

  respondWithJSON(w, 200, userDat)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
  token, err := auth.GetRefreshToken(r.Header)
  if err != nil {
    respondWithError(w, 401, "missing refresh token")
    return
  }

  dbToken, err := cfg.db.GetRefreshToken(r.Context(), token)
  if err != nil {
    respondWithError(w, 401, "token does not exist")
    return
  }

  if dbToken.RevokedAt.Valid {
    respondWithError(w, 401, "token expired")
    return
  }

  expTime := time.Unix(dbToken.ExpiresAt.Unix(), 0)
  if expTime.Before(time.Now()) {
    // expired, 
    respondWithError(w, 401, "expired or revoked token")
    return
  }

  // create new token & send ?
  user, err := cfg.db.GetUserFromRefreshToken(r.Context(), dbToken.Token)
  if err != nil {
    respondWithError(w, 500, "could not retrieve user")
    return
  }

  tok, err := auth.MakeJWT(user.ID, cfg.secret, 1 * time.Hour)
  if err != nil {
    respondWithError(w, 500, "could not create JWT token")
    return
  }


  type data struct {
    Token string `json:"token"`
  }

  dat := data{
    Token: tok,
  }

  respondWithJSON(w, 200, dat)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
  token, err := auth.GetRefreshToken(r.Header)
  if err != nil {
    respondWithError(w, 401, "missing refresh token")
    return
  }

  dbToken, err := cfg.db.GetRefreshToken(r.Context(), token)
  if err != nil {
    respondWithError(w, 401, "token does not exist")
    return
  }

  err = cfg.db.RevokeToken(r.Context(), dbToken.Token)
  if err != nil {
    respondWithError(w, 401, "could not revoke token")
  }

  w.WriteHeader(204) 
}

func (cfg *apiConfig) handlerUpdate(w http.ResponseWriter, r *http.Request) {
  token, err := auth.GetBearerToken(r.Header)
  if err != nil {
    respondWithError(w, 401, "invalid token")
    return
  }

  userID, err := auth.ValidateJWT(token, cfg.secret)
  if err != nil {
    respondWithError(w, 401, "invalid token")
    return
  }

  type parameters struct {
    Password string `json:"password"`
    Email string `json:"email"`
  }

  decoder := json.NewDecoder(r.Body)
  params := parameters{}
  err = decoder.Decode(&params)
  if err != nil {
    respondWithError(w, 401, "Something went wrong")
    return
  }

  hashed, err := auth.HashPassword(params.Password)
  if err != nil {
    respondWithError(w, 401, "could not change password")
    return
  }

  user, err := cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{
    ID: userID,
    Email: params.Email,
    HashedPassword: hashed,
  })
  if err != nil {
    respondWithError(w, 401, err.Error())
    return
  }

  userDat := User{
    ID: user.ID,
    CreatedAt: user.CreatedAt,
    UpdatedAt: user.UpdatedAt,
    Email: user.Email,
    IsChirpyRed: user.IsChirpyRed.Bool,
  }

  respondWithJSON(w, 200, userDat)
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
  token, err := auth.GetBearerToken(r.Header)
  if err != nil {
    respondWithError(w, 401, "invalid token")
    return
  }

  id := r.PathValue("chirpID")
  chirpID, err := uuid.Parse(id)
  if err != nil {
    respondWithError(w, 401, "invalid chirp id")
    return
  }

  userID, err := auth.ValidateJWT(token, cfg.secret)
  if err != nil {
    respondWithError(w, 401, "invalid token")
    return
  }

  user, err := cfg.db.GetUserById(r.Context(), userID)
  if err != nil {
    respondWithError(w, 401, "invalid token")
    return
  }


  chirpOwner, err := cfg.db.GetChirpById(r.Context(), chirpID)
  if err != nil {
    respondWithError(w, 401, "could not get chirp")
    return
  }

  if chirpOwner.UserID.UUID == user.ID {
    // user owns chirp, delete
    err := cfg.db.DeleteChirpById(r.Context(), chirpID)
    if err != nil {
      respondWithError(w, 401, "could not delete chirp")
      return
    }
    w.WriteHeader(204) 
    return
  }

  w.WriteHeader(403)
}

func (cfg *apiConfig) handlerUpgrade(w http.ResponseWriter, r *http.Request) {
  type data struct {
    UserID string `json:"user_id"`
  }

  type req struct {
    Event string `json:"event"`
    Data data 
  }

  decoder := json.NewDecoder(r.Body)
  reqData := req{}
  err := decoder.Decode(&reqData)
  if err != nil {
    respondWithError(w, 401, "Something went wrong")
    return
  }

  if reqData.Event != "user.upgraded" {
    w.WriteHeader(204)
    return
  }

  userID, err := uuid.Parse(reqData.Data.UserID)
  if err != nil {
    w.WriteHeader(401)
    return
  }

  _, err = cfg.db.UpgradeUserById(r.Context(), userID)
  if err != nil {
    w.WriteHeader(404)
    return
  }

  w.WriteHeader(204)
  
  

}

func main() {
    godotenv.Load()
    dbURL := os.Getenv("DB_URL")
    secret := os.Getenv("SECRET")
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
      log.Fatal(err) 
    }

    fmt.Println("Connected to DB.")

    dbQueries := database.New(db)

		mux := http.NewServeMux()	
		server := &http.Server{
			Addr: ":8080",
			Handler: mux,
		}
		apiConf := apiConfig{}
    apiConf.db = dbQueries
    apiConf.secret = secret
		mux.Handle("/app/", apiConf.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer((http.Dir("."))))))
		mux.HandleFunc("GET /api/healthz", readiness)
		mux.HandleFunc("GET /admin/metrics", apiConf.handlerMetrics)
		mux.HandleFunc("POST /admin/reset", apiConf.handlerReset)
    mux.HandleFunc("POST /api/users", apiConf.handlerUsers)
    mux.HandleFunc("POST /api/chirps", apiConf.handlerChirp)
    mux.HandleFunc("GET /api/chirps", apiConf.handlerGetChirps)
    mux.HandleFunc("GET /api/chirps/{chirpID}", apiConf.handlerGetChirp)
    mux.HandleFunc("POST /api/login", apiConf.handlerLogin)
    mux.HandleFunc("POST /api/refresh", apiConf.handlerRefresh)
    mux.HandleFunc("POST /api/revoke", apiConf.handlerRevoke)
    mux.HandleFunc("PUT /api/users", apiConf.handlerUpdate)
    mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiConf.handlerDeleteChirp)
    mux.HandleFunc("POST /api/polka/webhooks", apiConf.handlerUpgrade)
		server.ListenAndServe()
}