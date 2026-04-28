package cinema.base;

import cinema.utils.*;
import io.github.bonigarcia.wdm.WebDriverManager;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.chrome.*;
import org.testng.annotations.*;

import java.time.Duration;

public class BaseTest {
    protected WebDriver driver;
    public WebDriver getDriver() { return driver; }

    @BeforeClass(alwaysRun = true)
    public void setUp() {
        WebDriverManager.chromedriver().setup();
        ChromeOptions opts = new ChromeOptions();
        if (ConfigReader.getBool("headless")) opts.addArguments("--headless=new");
        opts.addArguments("--no-sandbox", "--disable-dev-shm-usage", "--window-size=1366,900");
        driver = new ChromeDriver(opts);
        driver.manage().timeouts().implicitlyWait(Duration.ofSeconds(ConfigReader.getInt("implicit.wait")));
    }

    @BeforeMethod(alwaysRun = true)
    public void openApp() { driver.get(ConfigReader.get("base.url")); }

    @AfterClass(alwaysRun = true)
    public void tearDown() {
        try { DBHelper.cleanup(); } catch (Exception e) { System.err.println("Cleanup warning: " + e.getMessage()); }
        if (driver != null) driver.quit();
    }
}
