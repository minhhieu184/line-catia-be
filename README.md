## NOTE

Because the initial requirements were relatively simple, I chose to use Redis instead of SQL to save time in development and optimize performance for the game. However, I recommend using SQL for easier scalability and maintenance in the long run.

## Getting Started

### Production

1. Before you start, let's be clear about environment variables.  
   The root `.env` file contains variables to be shared among services.  
   All leaf `.env` (from `.env.example`) files contain variables for their own service and needs to be configured before `docker-compose up`.

2. Load data from `quiz.csv` to database
   ```
   go run cmd/bank/main.go import
   ```

### Development

1. Create bot and get bot token.
2. Enter your bot token in `BOT_TOKEN` in file `env`
   ```
   BOT_TOKEN= <YOUR_BOT_TOKEN>
   ```
3. Enter community of game follow by `env.example`
4. Create `docker-compose.override.yml`
   ```shell
   cp docker-compose.development.yml docker-compose.override.yml
   ```
5. Enter mode of server.
   1. `MODE_SERVER=test` ignore unnecessary requirement.
   2. `MODE_SERVER=dev` allow api for test.
6. Load data from `quiz.csv` to database
   ```
   go run cmd/bank/main.go import
   ```
7. Run
   ```sh
   go run cmd/api/main.go server # run api server
   go run cmd/bot/main.go server # run bot
   ```

### Endpoints

```sh
# TOKEN FROM TMA
export TOKEN=''
export BASE_URL='https://par-dictionary-imported-pub.trycloudflare.com'

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/user/me

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia/session

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia/sessions/:id

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia/sessions/:id/end

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia/sessions/:id/assistance

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia/sessions/:id/answer -X POST -H 'content-type:application/json' -d '{"answer": 0}'

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia/next

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia/score

curl -H "authorization: Bearer $TOKEN" $BASE_URL/api/v1/game/catia/leaderboard

```
