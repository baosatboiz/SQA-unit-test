package cinema.utils;

import com.aventstack.extentreports.*;
import com.aventstack.extentreports.reporter.ExtentSparkReporter;
import org.openqa.selenium.*;
import org.testng.*;

import java.io.File;
import java.text.SimpleDateFormat;
import java.util.Date;

public class ExtentReportListener implements ITestListener {
    private static ExtentReports extent;
    private static final ThreadLocal<ExtentTest> testThread = new ThreadLocal<>();
    private static String reportPath;

    public static synchronized ExtentReports getInstance() {
        if (extent == null) {
            String ts = new SimpleDateFormat("yyyyMMdd-HHmmss").format(new Date());
            reportPath = "reports/extent-report-" + ts + ".html";
            new File("reports/screenshots").mkdirs();
            ExtentSparkReporter spark = new ExtentSparkReporter(reportPath);
            spark.config().setDocumentTitle("Cinema UI Test Report");
            spark.config().setReportName("Cinema Booking — Client Flow");
            spark.config().setTheme(com.aventstack.extentreports.reporter.configuration.Theme.STANDARD);
            extent = new ExtentReports();
            extent.attachReporter(spark);
            extent.setSystemInfo("OS", System.getProperty("os.name"));
            extent.setSystemInfo("Java", System.getProperty("java.version"));
            extent.setSystemInfo("Base URL", ConfigReader.get("base.url"));
        }
        return extent;
    }

    public static ExtentTest getTest() { return testThread.get(); }

    @Override public void onTestStart(ITestResult r) {
        testThread.set(getInstance().createTest(r.getMethod().getMethodName()));
    }
    @Override public void onTestSuccess(ITestResult r) {
        testThread.get().pass("Passed");
    }
    @Override public void onTestFailure(ITestResult r) {
        ExtentTest t = testThread.get();
        t.fail(r.getThrowable());
        Object inst = r.getInstance();
        if (inst instanceof cinema.base.BaseTest bt && bt.getDriver() != null) {
            try {
                String ts = new SimpleDateFormat("HHmmss-SSS").format(new Date());
                String img = "reports/screenshots/" + r.getMethod().getMethodName() + "-" + ts + ".png";
                File src = ((TakesScreenshot) bt.getDriver()).getScreenshotAs(OutputType.FILE);
                java.nio.file.Files.copy(src.toPath(), new File(img).toPath());
                t.addScreenCaptureFromPath("screenshots/" + new File(img).getName());
            } catch (Exception ignored) {}
        }
    }
    @Override public void onTestSkipped(ITestResult r) { testThread.get().skip(r.getThrowable()); }
    @Override public void onFinish(ITestContext c) { getInstance().flush(); }
}
