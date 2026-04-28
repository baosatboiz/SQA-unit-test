package cinema.scripts;

import org.apache.poi.ss.usermodel.*;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import java.io.FileOutputStream;
import java.io.File;

public class GenerateExcel {
    public static void main(String[] args) throws Exception {
        new File("test-data").mkdirs();
        try (Workbook wb = new XSSFWorkbook()) {
            // RegisterData
            Sheet s1 = wb.createSheet("RegisterData");
            String[] h1 = {"tcId","email","password","firstName","lastName","address","expectedResult","expectedMessage"};
            writeHeader(s1, h1);
            Object[][] d1 = {
                {"UI-01","qa_test_${ts}_01@example.com","Pass@1234","TestFirst","TestLast","123 Test St","PASS",""},
                {"UI-02","qa_test_existing@example.com","Pass@1234","Dup","Dup","456 Dup St","FAIL","exist"},
                {"UI-03","qa_test_${ts}_03@example.com","123","Weak","Pwd","789 Weak St","FAIL","password"},
            };
            writeRows(s1, d1);

            // LoginData
            Sheet s2 = wb.createSheet("LoginData");
            String[] h2 = {"tcId","email","password","expectedResult","expectedMessage"};
            writeHeader(s2, h2);
            Object[][] d2 = {
                {"UI-04","qa_test_login@example.com","Pass@1234","PASS",""},
                {"UI-05","qa_test_login@example.com","WrongPass1","FAIL","thất bại"},
            };
            writeRows(s2, d2);

            // BookingData
            Sheet s3 = wb.createSheet("BookingData");
            String[] h3 = {"tcId","email","password","movieIndex","showtimeIndex","seatStrategy","expectedResult"};
            writeHeader(s3, h3);
            Object[][] d3 = {
                {"UI-08","qa_test_booker@example.com","Pass@1234","0","0","FIRST_AVAILABLE","PASS"},
                {"UI-09","qa_test_booker@example.com","Pass@1234","0","0","BOOKED","FAIL"},
            };
            writeRows(s3, d3);

            // SearchData
            Sheet s4 = wb.createSheet("SearchData");
            String[] h4 = {"tcId","keyword","expectedMinResults"};
            writeHeader(s4, h4);
            Object[][] d4 = {
                {"UI-06","a","1"},
            };
            writeRows(s4, d4);

            try (FileOutputStream fos = new FileOutputStream("test-data/ClientFlowData.xlsx")) {
                wb.write(fos);
            }
            System.out.println("Created test-data/ClientFlowData.xlsx");
        }
    }

    static void writeHeader(Sheet s, String[] headers) {
        Row r = s.createRow(0);
        for (int i = 0; i < headers.length; i++) r.createCell(i).setCellValue(headers[i]);
    }

    static void writeRows(Sheet s, Object[][] data) {
        for (int r = 0; r < data.length; r++) {
            Row row = s.createRow(r + 1);
            for (int c = 0; c < data[r].length; c++) row.createCell(c).setCellValue(String.valueOf(data[r][c]));
        }
    }
}
