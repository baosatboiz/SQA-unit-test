# Cinema Booking System — QA Automation Design

**Ngày:** 2026-04-28
**Tác giả:** Trang
**Project under test:** `C:/ki8/Đảm bảo chất lượng phần mềm/cinema-project` (Cinema Booking microservices, Go + Node.js + React)
**QA workspace:** `C:/ki8/Đảm bảo chất lượng phần mềm/cinema-qa-automation`

---

## 1. Mục tiêu

Bài tập môn **Đảm bảo chất lượng phần mềm** yêu cầu 3 mảng test trên dự án Cinema Booking System:

1. **Functional / UI test** bằng Selenium WebDriver (không phải IDE) — auto càng nhiều system testcase càng tốt; dùng Excel/CSV cho test data; verify cả interface lẫn database; có rollback; sinh report.
2. **API test** bằng Postman — testcase cho từng API; verify cả interface lẫn database; có rollback; sinh report.
3. **Performance test** bằng JMeter — sinh testcase, script, chạy và sinh report cho môi trường mẫu.

Phạm vi đã chốt với người dùng:

| Module | Scope |
|---|---|
| Selenium UI | Chỉ Client flow (~10 testcase: register, login, browse, đặt vé, thanh toán, lịch sử) |
| Postman API | ~19 endpoint core flow (auth, movie, showtime/seat, booking, payment, user) |
| JMeter | 1 user journey end-to-end, 50–100 user song song |

---

## 2. Kiến trúc thư mục & công cụ

### 2.1 Cấu trúc thư mục

```
C:\ki8\Đảm bảo chất lượng phần mềm\cinema-qa-automation\
├── README.md
├── selenium-ui-test\
│   ├── pom.xml
│   ├── testng.xml
│   ├── run-selenium.bat
│   ├── src\test\java\cinema\
│   │   ├── base\BaseTest.java
│   │   ├── pages\
│   │   │   ├── LoginPage.java
│   │   │   ├── RegisterPage.java
│   │   │   ├── MovieListPage.java
│   │   │   ├── MovieDetailPage.java
│   │   │   ├── SeatSelectionPage.java
│   │   │   ├── PaymentPage.java
│   │   │   └── BookingHistoryPage.java
│   │   ├── tests\
│   │   │   ├── RegisterTest.java
│   │   │   ├── LoginTest.java
│   │   │   ├── MovieBrowseTest.java
│   │   │   ├── BookingFlowTest.java
│   │   │   └── BookingHistoryTest.java
│   │   └── utils\
│   │       ├── ExcelReader.java
│   │       ├── DBHelper.java
│   │       ├── ConfigReader.java
│   │       └── ExtentReportListener.java
│   ├── test-data\
│   │   └── ClientFlowData.xlsx
│   ├── config\config.properties
│   └── reports\               # ExtentReports HTML + screenshots
│
├── postman-api-test\
│   ├── collections\
│   │   └── Cinema-API.postman_collection.json
│   ├── environments\
│   │   └── Local.postman_environment.json
│   ├── data\
│   │   └── api-test-data.csv
│   ├── cleanup.sql
│   ├── run-newman.bat
│   └── reports\               # Newman htmlextra HTML
│
├── jmeter-perf-test\
│   ├── test-plans\
│   │   └── cinema-user-journey.jmx
│   ├── data\
│   │   └── users.csv
│   ├── seed-perf-users.sql
│   ├── cleanup.sql
│   ├── run-jmeter.bat
│   └── reports\               # JMeter HTML dashboard
│
└── docs\
    ├── test-cases-master.xlsx
    ├── final-report.md
    └── superpowers\specs\2026-04-28-cinema-qa-automation-design.md
```

### 2.2 Stack công cụ

| Module | Tool & version |
|---|---|
| Selenium | Java 17, Maven 3.9, TestNG 7.10, Selenium 4.x, WebDriverManager 5.x, Apache POI 5.x, ExtentReports 5.x, PostgreSQL JDBC 42.7.x |
| Postman | Postman Desktop, Newman CLI, newman-reporter-htmlextra |
| JMeter | Apache JMeter 5.6.3 |
| DB | PostgreSQL 15 (đã có trong cinema-project docker compose) |

### 2.3 Tiền điều kiện môi trường

- `cinema-project` chạy bằng `docker compose up -d`
- FE: `http://localhost:3000`
- API Gateway: `http://localhost:8000`
- DB: `localhost:5433/cinema_app`, user `postgres`, pass `Trang@051203`
- DB đã reset 1 lần để có dữ liệu mẫu (movies, showtimes, rooms, seats)
- Java 17, Maven, JMeter 5.6.3, Newman + htmlextra reporter đã cài

---

## 3. Test Cases

### 3.1 Selenium UI test cases (10 TC, Client flow)

| TC ID | Tên | Mô tả | Loại | Verify DB |
|---|---|---|---|---|
| UI-01 | Register thành công | Email/password/name hợp lệ → tài khoản được tạo | Positive | `users` |
| UI-02 | Register email đã tồn tại | Đăng ký với email đã có → hiện lỗi | Negative | – |
| UI-03 | Register password yếu | Password < 8 ký tự / không ký tự đặc biệt → lỗi validation | Negative | – |
| UI-04 | Login thành công | Email + password đúng → vào trang chủ | Positive | – |
| UI-05 | Login sai password | Hiện thông báo "Invalid credentials" | Negative | – |
| UI-06 | Browse & search movie | Vào danh sách phim, search theo tên → ra kết quả đúng | Positive | – |
| UI-07 | View movie detail & showtime | Click 1 phim → detail + showtime list | Positive | – |
| UI-08 | Đặt vé happy path | Login → movie → showtime → ghế → confirm → payment OK | E2E Positive | `bookings`, `payments` |
| UI-09 | Đặt ghế đã được người khác đặt | Cố chọn ghế status `BOOKED` → nút disabled hoặc lỗi | Negative | – |
| UI-10 | Xem lịch sử booking | Vào My Bookings → thấy booking vừa tạo ở UI-08 | Positive | `bookings` |

Mỗi TC có Pre-condition · Steps · Expected · Actual · Status — viết bằng TestNG `@Test`, log từng step vào ExtentReports, screenshot khi fail.

### 3.2 Postman API test cases (~19 TC)

| Group | Endpoint | Test cases |
|---|---|---|
| Auth (5) | `POST /api/v1/auth/register` | TC-API-01 valid · TC-API-02 duplicate email · TC-API-03 invalid email format |
| | `POST /api/v1/auth/login` | TC-API-04 valid · TC-API-05 wrong password |
| Movie (3) | `GET /api/v1/movies` | TC-API-06 list 200 · TC-API-07 pagination |
| | `GET /api/v1/movies/:id` | TC-API-08 not found 404 |
| Showtime/Seat (3) | `GET /api/v1/showtimes?movie_id=` | TC-API-09 list theo movie |
| | `GET /api/v1/showtimes/:id/seats` | TC-API-10 seat map · TC-API-11 invalid showtime id |
| Booking (4) | `POST /api/v1/bookings` | TC-API-12 create OK · TC-API-13 ghế đã đặt 409 · TC-API-14 unauthorized 401 |
| | `GET /api/v1/bookings/me` | TC-API-15 list của user |
| Payment (2) | `POST /api/v1/payments` | TC-API-16 create OK |
| | `POST /api/v1/payments/:id/confirm` | TC-API-17 confirm OK |
| User (2) | `GET /api/v1/users/me` | TC-API-18 profile |
| | `PUT /api/v1/users/me` | TC-API-19 update name/phone |

Mỗi TC có `pm.test()` assertions: status code, response schema, các field key (`id`, `email`, `status`…), `responseTime < 2000ms`. Negative case kiểm tra error message.

### 3.3 JMeter test plan (1 user journey)

**Scenario** Cinema booking user journey:
```
[CSV: users.csv] → Login → GET /movies → GET /showtimes → GET /seats →
POST /bookings → POST /payments → POST /payments/:id/confirm
```

| Stage | Threads | Ramp-up | Loop | Mục đích |
|---|---|---|---|---|
| Warm-up | 10 | 30s | 1 | Khởi động JVM, kiểm tra script |
| Normal load | 50 | 60s | 5 | Tải bình thường |
| Peak load | 100 | 60s | 5 | Tải đỉnh |

**Listeners** Aggregate Report, Summary Report, View Results Tree (chỉ debug), HTML Dashboard.
**Assertion** response time `< 2000ms` (95th percentile target), error rate `< 1%`.
**Metrics** avg / min / max response time, throughput (req/sec), error %, 90/95/99 percentile.

---

## 4. Test Data Strategy

### 4.1 Excel cho Selenium

File `selenium-ui-test/test-data/ClientFlowData.xlsx`:

| Sheet | Cột | Dùng cho |
|---|---|---|
| `RegisterData` | tcId, email, password, fullName, phone, expectedResult, expectedMessage | UI-01, 02, 03 |
| `LoginData` | tcId, email, password, expectedResult, expectedMessage | UI-04, 05 |
| `BookingData` | tcId, movieName, showtimeIndex, seatCode, paymentMethod, expectedResult | UI-08, 09 |
| `SearchData` | tcId, keyword, expectedMinResults | UI-06 |

Đọc bằng Apache POI (`XSSFWorkbook`) qua `ExcelReader.java`. Mỗi row = 1 lần chạy test (TestNG `@DataProvider`). Email test dùng prefix `qa_test_<timestamp>@example.com` để dễ rollback.

### 4.2 CSV cho Newman

`postman-api-test/data/api-test-data.csv` — cột `tcId, email, password, expectedStatus, expectedField, …` cho data-driven request.

### 4.3 CSV cho JMeter

`jmeter-perf-test/data/users.csv` — 100 user đã được seed sẵn:
```
email,password,userId
qa_perf_user_001@test.com,Pass@1234,
...
qa_perf_user_100@test.com,Pass@1234,
```

`CSV Data Set Config` của JMeter đọc từng dòng, mỗi thread 1 user.

---

## 5. Rollback Strategy

Nguyên tắc dùng prefix data + cleanup script — không truncate bảng để giữ dữ liệu seed gốc.

| Module | Cách rollback |
|---|---|
| Selenium | `BaseTest.@AfterClass` gọi `DBHelper.cleanup()`: `DELETE` payments → bookings → users theo FK order, `WHERE email LIKE 'qa_test_%'` |
| Postman/Newman | Sau khi Newman chạy xong, `run-newman.bat` chạy `psql -h localhost -p 5433 -U postgres -d cinema_app -f cleanup.sql` xóa data prefix `qa_api_%` |
| JMeter | `run-jmeter.bat` chạy `psql -f cleanup.sql` xóa booking/payment có user email `qa_perf_%`. Giữ users `qa_perf_%` để chạy lại lần sau |

Cleanup script là **idempotent** — chạy lại bất kỳ lúc nào không gây lỗi.

### Seed data 1 lần duy nhất

`jmeter-perf-test/seed-perf-users.sql` INSERT 100 user `qa_perf_user_001..100` với password bcrypt hash sẵn cho `Pass@1234`. Chạy thủ công 1 lần bằng `psql`.

### DB verify (Selenium)

`DBHelper.java` cung cấp:
- `boolean userExists(String email)` — verify UI-01
- `int countBookings(String email)` — verify UI-08, UI-10
- `String getBookingStatus(long bookingId)` — verify `CONFIRMED`
- `String getPaymentStatus(long bookingId)` — verify `PAID`

Connection JDBC `jdbc:postgresql://localhost:5433/cinema_app`, credentials đọc từ `config.properties` (không hardcode).

---

## 6. Reporting

### 6.1 Selenium — ExtentReports HTML

Output `selenium-ui-test/reports/extent-report-<timestamp>.html`. Listener `ExtentReportListener implements ITestListener` (TestNG) tự log step và chụp screenshot khi fail. Báo cáo gồm dashboard pass/fail/skip, per-test step + screenshot, system info, filter, timeline biểu đồ.

### 6.2 Postman — Newman + htmlextra

Output `postman-api-test/reports/newman-report-<timestamp>.html`. Command:
```bat
newman run collections/Cinema-API.postman_collection.json ^
  -e environments/Local.postman_environment.json ^
  -d data/api-test-data.csv ^
  -r htmlextra,cli ^
  --reporter-htmlextra-export reports/newman-report.html ^
  --reporter-htmlextra-title "Cinema API Test Report" ^
  --reporter-htmlextra-darkTheme
```

### 6.3 JMeter — HTML Dashboard

Output `jmeter-perf-test/reports/dashboard-<timestamp>/index.html`. Command:
```bat
jmeter -n -t test-plans/cinema-user-journey.jmx ^
  -l reports/results.jtl ^
  -e -o reports/dashboard-%timestamp%
```

Dashboard có APDEX, statistics (avg/percentile/throughput/error%), charts (response time over time, hits/sec, active threads), errors table, top slowest requests.

### 6.4 Final report tổng hợp

`docs/final-report.md` viết tay sau khi chạy 3 module xong, gồm:
1. Tóm tắt — môi trường, ngày chạy, tester
2. Selenium UI test — bảng 10 TC, link extent-report.html, ảnh dashboard
3. API test — bảng ~19 TC, link newman-report.html, ảnh summary
4. Performance test — kết luận throughput, response time, breaking point, link dashboard, ảnh biểu đồ
5. Defects found — list bug, severity, screenshot
6. Kết luận & khuyến nghị

Có thể export PDF nộp giảng viên.

---

## 7. Execution Flow

### 7.1 Pre-flight (1 lần duy nhất)

```
1. cd cinema-project && docker compose up -d
2. Reset DB lần đầu (theo HUONG_DAN_CHAY.md)
3. Cài Java 17, Maven 3.9, JMeter 5.6.3, npm i -g newman newman-reporter-htmlextra
4. Seed 100 perf user: psql ... -f jmeter-perf-test/seed-perf-users.sql
5. cd cinema-qa-automation/selenium-ui-test && mvn clean install
```

### 7.2 Mỗi module 1 file `.bat` chạy độc lập

| File | Làm gì |
|---|---|
| `selenium-ui-test/run-selenium.bat` | `mvn clean test -DsuiteXmlFile=testng.xml` → ExtentReport |
| `postman-api-test/run-newman.bat` | Newman + CSV → htmlextra → `psql -f cleanup.sql` |
| `jmeter-perf-test/run-jmeter.bat` | JMeter non-GUI → dashboard HTML → `psql -f cleanup.sql` |

### 7.3 Lifecycle 1 testcase Selenium

```
@BeforeClass   init WebDriver, ExtentReports, DBHelper
@BeforeMethod  open browser → http://localhost:3000, log testcase
@Test          đọc Excel via DataProvider → POM steps → assert UI + DB →
               log step, screenshot khi fail
@AfterMethod   log status, attach screenshot
@AfterClass    DBHelper.cleanup() xóa qa_test_%, flush report, close driver
```

### 7.4 Lifecycle 1 collection Newman

```
Pre-request (collection): set baseUrl, timestamp variable
Per request:
  Pre-request (folder): tạo dynamic email qa_api_{{$timestamp}}@…
  Request: gửi
  Tests: assertions (status, schema, fields, time) + lưu token/id
Post-collection: psql cleanup.sql
```

### 7.5 Lifecycle JMeter user journey

```
setUp Thread Group: ping API gateway health

Main Thread Group (50 → 100 users):
  CSV Data Set Config (users.csv) đọc email/password
  ├── POST /auth/login         → JSON Extractor token
  ├── GET  /movies             → extract movieId
  ├── GET  /showtimes?movie_id → extract showtimeId
  ├── GET  /showtimes/:id/seats → extract seatId
  ├── POST /bookings           → extract bookingId
  ├── POST /payments           → extract paymentId
  └── POST /payments/:id/confirm
  Gaussian Random Timer (mean 1000ms, dev 500ms)
  Response Assertion: time < 2000ms

tearDown Thread Group: trigger psql cleanup
```

### 7.6 Trình tự khi demo / nộp bài

```
1. docker compose up -d                            (cinema project)
2. psql -f seed-perf-users.sql                     (1 lần)
3. cd selenium-ui-test && run-selenium.bat         (~5–8 phút)
4. cd postman-api-test && run-newman.bat           (~2 phút)
5. cd jmeter-perf-test && run-jmeter.bat           (~5–10 phút)
6. Mở 3 file report, chụp ảnh, viết final-report.md
```

### 7.7 Error handling & edge cases

| Tình huống | Xử lý |
|---|---|
| Cinema project chưa chạy | Mỗi `run-*.bat` `curl http://localhost:8000/health` trước, fail-fast nếu down |
| DB connection fail | `DBHelper` log lỗi chi tiết, test fail nhanh không treo |
| Browser/driver chưa cài | WebDriverManager auto-download chromedriver |
| Test bị bỏ dở | Cleanup script idempotent, chạy lại bất kỳ lúc nào |
| Element UI không xuất hiện | ExplicitWait 10s + screenshot + fail testcase |

---

## 8. Out of Scope

- Admin flow UI test (đã loại ở Q1)
- Mobile app (Appium) — đề bài liệt kê là tùy chọn
- Chatbot, Notification (WebSocket), Analytics, Worker service — không nằm trong core booking flow
- Blockchain payment (Ethereum) — quá phức tạp với phạm vi môn học
- CI/CD pipeline (chạy local là đủ)
- Stress test ramp-to-failure (đã chốt PA1, JMeter chỉ làm load test)

---

## 9. Tiêu chí thành công

- Selenium chạy 10/10 TC ổn định, ExtentReport HTML mở được, có screenshot khi fail
- Newman chạy 19/19 TC, htmlextra report mở được, các assertion pass đúng kỳ vọng
- JMeter dashboard có đủ APDEX, response time chart, throughput chart, error table
- Cleanup chạy xong DB không còn data prefix `qa_test_%` / `qa_api_%` / `qa_perf_%` (booking/payment)
- `docs/final-report.md` đầy đủ 6 mục, có thể export PDF nộp giảng viên
