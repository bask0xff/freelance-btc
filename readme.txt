http://localhost:5180/


docker compose down
docker compose up -d

----
delete DB data and new migrations:

docker compose down -v
docker compose up -d --build

#флаг --no-cache подстрахует от случайно закэшированных старых слоёв Docker.
docker compose build --no-cache && docker compose up -d

#теперь там должно быть "running database migrations...", "migrations OK", "bitcoind RPC OK...", "listening on :8080" — без единого упоминания mysql.
docker compose logs backend --tail 50