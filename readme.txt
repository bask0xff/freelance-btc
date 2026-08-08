docker compose down
docker compose up -d

----
delete DB data and new migrations:

docker compose down -v
docker compose up -d --build
