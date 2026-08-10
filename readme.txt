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


https://gemini.google.com/app/ef6513687c686170 (С Б)
https://claude.ai/chat/68a75c73-9a4d-4db8-b1ad-d1b452b4f2d5 (sergon...)

----------

d:\Projects\2026\freelance-btc\backend>go test -v ./...
?       freelance-btc/cmd/api   [no test files]
?       freelance-btc/internal/config   [no test files]
?       freelance-btc/internal/db       [no test files]
=== RUN   TestApply_OrderNotFound
--- PASS: TestApply_OrderNotFound (0.00s)
=== RUN   TestApply_OrderNotOpen
--- PASS: TestApply_OrderNotOpen (0.00s)
=== RUN   TestApply_Success
--- PASS: TestApply_Success (0.00s)
=== RUN   TestApply_WithProposedAmount
--- PASS: TestApply_WithProposedAmount (0.00s)
=== RUN   TestWithdraw_NotFoundOrNotYours
--- PASS: TestWithdraw_NotFoundOrNotYours (0.00s)
=== RUN   TestWithdraw_Success
--- PASS: TestWithdraw_Success (0.00s)
PASS
ok      freelance-btc/internal/application      1.027s
=== RUN   TestGenerateAndParseToken
--- PASS: TestGenerateAndParseToken (0.00s)
=== RUN   TestParseToken_WrongSecret
--- PASS: TestParseToken_WrongSecret (0.00s)
=== RUN   TestParseToken_Expired
--- PASS: TestParseToken_Expired (0.00s)
=== RUN   TestParseToken_Garbage
--- PASS: TestParseToken_Garbage (0.00s)
PASS
ok      freelance-btc/internal/auth     1.355s
=== RUN   TestStatusFor
=== RUN   TestStatusFor/zero_confirmations
=== RUN   TestStatusFor/below_threshold
=== RUN   TestStatusFor/exactly_threshold
=== RUN   TestStatusFor/above_threshold
=== RUN   TestStatusFor/threshold_zero_always_confirmed
--- PASS: TestStatusFor (0.00s)
    --- PASS: TestStatusFor/zero_confirmations (0.00s)
    --- PASS: TestStatusFor/below_threshold (0.00s)
    --- PASS: TestStatusFor/exactly_threshold (0.00s)
    --- PASS: TestStatusFor/above_threshold (0.00s)
    --- PASS: TestStatusFor/threshold_zero_always_confirmed (0.00s)
PASS
ok      freelance-btc/internal/bitcoin  0.985s
=== RUN   TestCreateOrder_Success
--- PASS: TestCreateOrder_Success (0.00s)
=== RUN   TestHireFreelancer_Success
--- PASS: TestHireFreelancer_Success (0.01s)
=== RUN   TestHireFreelancer_NotOwner
--- PASS: TestHireFreelancer_NotOwner (0.00s)
=== RUN   TestHireFreelancer_OrderNotOpen
--- PASS: TestHireFreelancer_OrderNotOpen (0.00s)
PASS
ok      freelance-btc/internal/order    1.505s
=== RUN   TestRegister_Success
--- PASS: TestRegister_Success (0.12s)
=== RUN   TestRegister_InvalidEmail
--- PASS: TestRegister_InvalidEmail (0.00s)
=== RUN   TestRegister_ShortPassword
--- PASS: TestRegister_ShortPassword (0.00s)
=== RUN   TestRegister_InvalidRole
--- PASS: TestRegister_InvalidRole (0.00s)
=== RUN   TestRegister_DuplicateEmail
--- PASS: TestRegister_DuplicateEmail (0.11s)
=== RUN   TestAuthenticate_WrongPassword
--- PASS: TestAuthenticate_WrongPassword (0.00s)
=== RUN   TestAuthenticate_Success
--- PASS: TestAuthenticate_Success (0.00s)
PASS
ok      freelance-btc/internal/user     1.102s

d:\Projects\2026\freelance-btc\backend>


--------------

