package cinema.scripts;

import org.apache.poi.ss.usermodel.*;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import java.io.FileOutputStream;
import java.io.File;

public class GenerateMasterTC {
    public static void main(String[] args) throws Exception {
        new File("../docs").mkdirs();
        try (Workbook wb = new XSSFWorkbook()) {
            Sheet s = wb.createSheet("MasterTestCases");
            String[] h = {"TC ID","Module","Tên","Loại","Mô tả","Verify DB"};
            Row hr = s.createRow(0);
            for (int i = 0; i < h.length; i++) hr.createCell(i).setCellValue(h[i]);

            Object[][] data = {
                // Selenium
                {"UI-01","Selenium","Register thành công","Positive","Email/password/firstName/lastName/address hợp lệ → tài khoản tạo, redirect /verify","users"},
                {"UI-02","Selenium","Register email đã tồn tại","Negative","Đăng ký email đã có → hiện lỗi",""},
                {"UI-03","Selenium","Register password yếu","Negative","Password ngắn → lỗi validation",""},
                {"UI-04","Selenium","Login thành công","Positive","Email + password đúng → vào trang chủ",""},
                {"UI-05","Selenium","Login sai password","Negative","Hiện thông báo lỗi",""},
                {"UI-06","Selenium","Browse & search movie","Positive","Vào /movies, lọc → thấy ≥1 kết quả",""},
                {"UI-07","Selenium","View movie showtime","Positive","Click 1 phim → modal showtime mở",""},
                {"UI-08","Selenium","Đặt vé happy path","E2E Positive","Login → movie → showtime → ghế → đến trang payment","bookings"},
                {"UI-09","Selenium","Ghế đã đặt disabled","Negative","Ghế status BOOKED không click được",""},
                {"UI-10","Selenium","Xem lịch sử booking","Positive","/booking-history hiển thị booking vừa tạo","bookings"},
                // Postman
                {"TC-API-01","Postman","Register valid","Positive","POST /auth/register status 201",""},
                {"TC-API-02","Postman","Register duplicate","Negative","Email tồn tại → 409/400",""},
                {"TC-API-03","Postman","Register invalid email","Negative","Format sai → 400/422",""},
                {"TC-API-04","Postman","Login valid","Positive","Có token trả về",""},
                {"TC-API-05","Postman","Login wrong password","Negative","401",""},
                {"TC-API-06","Postman","List movies","Positive","GET /movies status 200",""},
                {"TC-API-07","Postman","Pagination","Positive","page=1&per_page=5",""},
                {"TC-API-08","Postman","Movie not found","Negative","ID không tồn tại → 404",""},
                {"TC-API-09","Postman","List showtimes by movie","Positive","Lấy theo movie_id",""},
                {"TC-API-10","Postman","Seat map","Positive","Có array seats",""},
                {"TC-API-11","Postman","Invalid showtime id","Negative","400/404",""},
                {"TC-API-12","Postman","Create booking","Positive","Status 201",""},
                {"TC-API-13","Postman","Booking duplicate seat","Negative","409",""},
                {"TC-API-14","Postman","Booking unauthorized","Negative","401",""},
                {"TC-API-15","Postman","List my bookings","Positive","Có booking vừa tạo",""},
                {"TC-API-16","Postman","Create payment","Positive","Status 201",""},
                {"TC-API-17","Postman","Confirm payment","Positive","Status 200",""},
                {"TC-API-18","Postman","Get profile","Positive","Có email",""},
                {"TC-API-19","Postman","Update profile","Positive","Cập nhật firstName/lastName/address",""},
                // JMeter
                {"JM-01","JMeter","Cinema user journey","Load","50 user × 5 loop, ramp-up 60s, journey login→movies→showtimes→seats→booking→payment",""},
            };
            for (int r = 0; r < data.length; r++) {
                Row row = s.createRow(r + 1);
                for (int c = 0; c < data[r].length; c++) row.createCell(c).setCellValue(String.valueOf(data[r][c]));
            }

            try (FileOutputStream fos = new FileOutputStream("../docs/test-cases-master.xlsx")) {
                wb.write(fos);
            }
            System.out.println("Created ../docs/test-cases-master.xlsx");
        }
    }
}
