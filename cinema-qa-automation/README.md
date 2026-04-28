# Cinema QA Automation

3 module test automation cho dự án `cinema-project`:
- `selenium-ui-test/` — Selenium WebDriver, Java + TestNG, ExtentReports
- `postman-api-test/` — Postman + Newman + htmlextra
- `jmeter-perf-test/` — Apache JMeter 5.6.3

## Tiền điều kiện

1. `cinema-project` đang chạy: `cd ../cinema-project && docker compose up -d`
2. Cài: Java 17, Maven 3.9, JMeter 5.6.3, Newman (`npm i -g newman newman-reporter-htmlextra`), PostgreSQL client (`psql`)
3. Seed perf users (1 lần): `psql -h localhost -p 5433 -U postgres -d cinema_app -f jmeter-perf-test/seed-perf-users.sql`

## Chạy

```bash
cd selenium-ui-test  && run-selenium.bat
cd postman-api-test  && run-newman.bat
cd jmeter-perf-test  && run-jmeter.bat
```

Mở report HTML trong từng `reports/` tương ứng. Tổng hợp ở `docs/final-report.md`.
