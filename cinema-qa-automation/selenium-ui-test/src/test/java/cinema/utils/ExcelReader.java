package cinema.utils;

import org.apache.poi.ss.usermodel.*;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;

import java.io.FileInputStream;
import java.util.*;

public class ExcelReader {
    public static List<Map<String, String>> read(String filePath, String sheetName) {
        List<Map<String, String>> rows = new ArrayList<>();
        try (FileInputStream fis = new FileInputStream(filePath);
             Workbook wb = new XSSFWorkbook(fis)) {
            Sheet sheet = wb.getSheet(sheetName);
            if (sheet == null) throw new RuntimeException("Sheet not found: " + sheetName);
            Row header = sheet.getRow(0);
            List<String> headers = new ArrayList<>();
            for (Cell c : header) headers.add(c.getStringCellValue().trim());

            DataFormatter df = new DataFormatter();
            for (int r = 1; r <= sheet.getLastRowNum(); r++) {
                Row row = sheet.getRow(r);
                if (row == null) continue;
                Map<String, String> map = new LinkedHashMap<>();
                for (int c = 0; c < headers.size(); c++) {
                    Cell cell = row.getCell(c, Row.MissingCellPolicy.CREATE_NULL_AS_BLANK);
                    map.put(headers.get(c), df.formatCellValue(cell).trim());
                }
                rows.add(map);
            }
        } catch (Exception e) {
            throw new RuntimeException("Failed to read Excel: " + filePath, e);
        }
        return rows;
    }

    public static Object[][] toDataProvider(String filePath, String sheetName) {
        List<Map<String, String>> rows = read(filePath, sheetName);
        Object[][] data = new Object[rows.size()][1];
        for (int i = 0; i < rows.size(); i++) data[i][0] = rows.get(i);
        return data;
    }
}
