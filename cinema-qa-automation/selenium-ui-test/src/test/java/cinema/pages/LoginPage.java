package cinema.pages;

import cinema.utils.ConfigReader;
import org.openqa.selenium.By;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedConditions;
import org.openqa.selenium.support.ui.WebDriverWait;

import java.time.Duration;
import java.util.List;

/**
 * Page Object for /login
 *
 * Real FE structure (LoginPage.jsx):
 *   <input type="email">                   — no name attr
 *   <input type="password">                — no name attr
 *   <button type="submit">ĐĂNG NHẬP</button>
 *   Error div: div containing class text-red-300 or bg-red-900
 */
public class LoginPage {

    private final WebDriver driver;
    private final WebDriverWait wait;

    // Locators
    private static final By EMAIL_INPUT    = By.cssSelector("input[type='email']");
    private static final By PASSWORD_INPUT = By.cssSelector("input[type='password']");
    private static final By SUBMIT_BUTTON  = By.cssSelector("button[type='submit']");
    // Tolerant error selector — matches either text-red-300 span or bg-red-900/50 wrapper div
    private static final By ERROR_MSG      = By.xpath(
        "//*[contains(@class,'text-red-300') or contains(@class,'bg-red-900')]"
    );

    public LoginPage(WebDriver driver) {
        this.driver = driver;
        this.wait   = new WebDriverWait(driver, Duration.ofSeconds(15));
    }

    /** Navigate to the login page. */
    public LoginPage open() {
        String base = ConfigReader.get("base.url");
        driver.get(base + "/login");
        wait.until(ExpectedConditions.visibilityOfElementLocated(EMAIL_INPUT));
        return this;
    }

    /** Type into the email field. */
    public LoginPage enterEmail(String email) {
        WebElement el = wait.until(ExpectedConditions.visibilityOfElementLocated(EMAIL_INPUT));
        el.clear();
        el.sendKeys(email);
        return this;
    }

    /** Type into the password field. */
    public LoginPage enterPassword(String password) {
        WebElement el = wait.until(ExpectedConditions.visibilityOfElementLocated(PASSWORD_INPUT));
        el.clear();
        el.sendKeys(password);
        return this;
    }

    /** Click the ĐĂNG NHẬP submit button. */
    public void submit() {
        wait.until(ExpectedConditions.elementToBeClickable(SUBMIT_BUTTON)).click();
    }

    /**
     * Returns the visible error message text, or empty string if no error is shown.
     * Waits up to 5 s for the error element to appear.
     */
    public String getErrorMessage() {
        try {
            WebDriverWait shortWait = new WebDriverWait(driver, Duration.ofSeconds(5));
            List<WebElement> errors = shortWait.until(
                ExpectedConditions.visibilityOfAllElementsLocatedBy(ERROR_MSG)
            );
            return errors.isEmpty() ? "" : errors.get(0).getText().trim();
        } catch (Exception e) {
            return "";
        }
    }

    /**
     * Returns true if the current URL is "/" (home), indicating a successful login.
     */
    public boolean isLoggedIn() {
        try {
            wait.until(driver -> {
                String url = driver.getCurrentUrl();
                return url != null && (url.endsWith("/") || url.matches(".*/\\?.*") ||
                    (!url.contains("/login") && !url.contains("/register")));
            });
            String url = driver.getCurrentUrl();
            return url != null && !url.contains("/login");
        } catch (Exception e) {
            return false;
        }
    }
}
