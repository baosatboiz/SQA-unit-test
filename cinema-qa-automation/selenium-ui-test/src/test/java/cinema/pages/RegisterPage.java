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
 * Page Object for /register
 *
 * Real FE fields (RegisterPage.jsx):
 *   <input name="email">
 *   <input name="password">
 *   <input name="confirmPassword">
 *   <input name="firstName">
 *   <input name="lastName">
 *   <input name="address">
 *   NOTE: no phone field exists in the actual FE form.
 *
 *   <button type="submit">ĐĂNG KÝ</button>
 *   Success: green div  bg-green-900/50 — "Đăng ký thành công..."
 *   Error  : red div    bg-red-900/50
 *   On success navigates to /verify?email=...
 */
public class RegisterPage {

    private final WebDriver driver;
    private final WebDriverWait wait;

    // Locators
    private static final By EMAIL_INPUT            = By.cssSelector("input[name='email']");
    private static final By PASSWORD_INPUT         = By.cssSelector("input[name='password']");
    private static final By CONFIRM_PASSWORD_INPUT = By.cssSelector("input[name='confirmPassword']");
    private static final By FIRST_NAME_INPUT       = By.cssSelector("input[name='firstName']");
    private static final By LAST_NAME_INPUT        = By.cssSelector("input[name='lastName']");
    private static final By ADDRESS_INPUT          = By.cssSelector("input[name='address']");
    private static final By SUBMIT_BUTTON          = By.cssSelector("button[type='submit']");
    private static final By ERROR_MSG              = By.xpath(
        "//*[contains(@class,'bg-red-900') or contains(@class,'text-red-')]"
    );
    private static final By SUCCESS_MSG            = By.xpath(
        "//*[contains(@class,'bg-green-900') or contains(@class,'text-green-')]"
    );

    public RegisterPage(WebDriver driver) {
        this.driver = driver;
        this.wait   = new WebDriverWait(driver, Duration.ofSeconds(15));
    }

    /** Navigate to the registration page. */
    public RegisterPage open() {
        String base = ConfigReader.get("base.url");
        driver.get(base + "/register");
        wait.until(ExpectedConditions.visibilityOfElementLocated(EMAIL_INPUT));
        return this;
    }

    /**
     * Fill all registration form fields.
     *
     * @param email           account email
     * @param password        desired password
     * @param confirmPassword password confirmation (must match password)
     * @param firstName       first name
     * @param lastName        last name
     * @param address         address
     */
    public RegisterPage fill(String email, String password, String confirmPassword,
                             String firstName, String lastName, String address) {
        fillField(EMAIL_INPUT,            email);
        fillField(PASSWORD_INPUT,         password);
        fillField(CONFIRM_PASSWORD_INPUT, confirmPassword);
        fillField(FIRST_NAME_INPUT,       firstName);
        fillField(LAST_NAME_INPUT,        lastName);
        fillField(ADDRESS_INPUT,          address);
        return this;
    }

    /** Click the ĐĂNG KÝ submit button. */
    public void submit() {
        wait.until(ExpectedConditions.elementToBeClickable(SUBMIT_BUTTON)).click();
    }

    /**
     * Returns the visible error message text, or empty string if none found within 5 s.
     */
    public String getError() {
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
     * Returns true if registration succeeded — either a green success banner is visible
     * or the browser has navigated to /verify.
     */
    public boolean isSuccess() {
        try {
            WebDriverWait shortWait = new WebDriverWait(driver, Duration.ofSeconds(8));
            // Check for redirect to /verify page first
            shortWait.until(driver -> {
                String url = driver.getCurrentUrl();
                List<WebElement> successes = driver.findElements(SUCCESS_MSG);
                return (url != null && url.contains("/verify")) ||
                       successes.stream().anyMatch(WebElement::isDisplayed);
            });
            String url = driver.getCurrentUrl();
            return (url != null && url.contains("/verify")) ||
                   driver.findElements(SUCCESS_MSG).stream().anyMatch(WebElement::isDisplayed);
        } catch (Exception e) {
            return false;
        }
    }

    // ── helpers ─────────────────────────────────────────────────────────────

    private void fillField(By locator, String value) {
        WebElement el = wait.until(ExpectedConditions.visibilityOfElementLocated(locator));
        el.clear();
        el.sendKeys(value);
    }
}
