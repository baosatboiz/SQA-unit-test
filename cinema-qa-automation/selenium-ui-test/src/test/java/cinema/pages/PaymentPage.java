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
 * Page Object for /booking/:bookingId/payment (PaymentPage.jsx + PaymentMethods.jsx)
 *
 * Route: /booking/{bookingId}/payment
 *
 * Payment methods available (PaymentMethods.jsx):
 *   - "Chuyển khoản VND"  (default for online; button text contains "Chuyển khoản VND")
 *   - "Crypto"            (button text contains "Crypto")
 *
 * Success state:
 *   - PaymentMethods renders a green div:
 *       <div class="mb-6 p-4 bg-green-900/50 border border-green-600 ...">Thanh toán thành công!</div>
 *   - After success, navigates to /booking-success?bookingId=...
 *
 * Booking info section (PaymentPage.jsx):
 *   - <h3>Thông tin đặt vé</h3>
 *   - Booking ID, total amount, status, created_at
 *
 * NOTE: For automated tests the QR / bank-transfer flow cannot be fully automated
 *       (requires external bank action). The confirm method below demonstrates selecting
 *       a payment method; actual payment completion in tests relies on test-environment
 *       hooks or a mock payment service.
 *
 * TODO: If a test-environment "instant confirm" button exists, add its locator here.
 */
public class PaymentPage {

    private final WebDriver driver;
    private final WebDriverWait wait;

    // Payment method tabs
    private static final By VND_TAB    = By.xpath(
        "//button[contains(normalize-space(),'Chuyển khoản VND')]"
    );
    private static final By CRYPTO_TAB = By.xpath(
        "//button[contains(normalize-space(),'Crypto')]"
    );

    // Page heading
    private static final By PAGE_HEADING = By.xpath(
        "//h1[contains(normalize-space(),'Thanh toán')]"
    );

    // Booking info section heading
    private static final By BOOKING_INFO_HEADING = By.xpath(
        "//h3[contains(normalize-space(),'Thông tin đặt vé')]"
    );

    // Payment success indicator (PaymentMethods.jsx green banner)
    private static final By SUCCESS_BANNER = By.xpath(
        "//*[contains(@class,'bg-green-900') or contains(@class,'bg-green-50')]" +
        "[.//*[contains(normalize-space(),'Thanh toán thành công')] " +
        " or contains(normalize-space(),'Thanh toán thành công')]"
    );

    // Loading spinner (when page is initializing)
    private static final By LOADING_SPINNER = By.cssSelector(
        "div.animate-spin"
    );

    public PaymentPage(WebDriver driver) {
        this.driver = driver;
        this.wait   = new WebDriverWait(driver, Duration.ofSeconds(20));
    }

    /**
     * Navigate directly to the payment page for a given booking ID.
     */
    public PaymentPage open(String bookingId) {
        String base = ConfigReader.get("base.url");
        driver.get(base + "/booking/" + bookingId + "/payment");
        waitForLoad();
        return this;
    }

    /**
     * Waits for the page to finish loading (spinner disappears, heading appears).
     */
    public PaymentPage waitForLoad() {
        // Wait for spinner to disappear (if present)
        try {
            WebDriverWait spinWait = new WebDriverWait(driver, Duration.ofSeconds(5));
            spinWait.until(ExpectedConditions.visibilityOfElementLocated(LOADING_SPINNER));
            wait.until(ExpectedConditions.invisibilityOfElementLocated(LOADING_SPINNER));
        } catch (Exception ignored) {
            // Spinner may not appear at all for fast responses
        }
        wait.until(ExpectedConditions.visibilityOfElementLocated(PAGE_HEADING));
        return this;
    }

    /**
     * Selects the "Chuyển khoản VND" payment method tab.
     */
    public PaymentPage selectVndTransfer() {
        wait.until(ExpectedConditions.elementToBeClickable(VND_TAB)).click();
        return this;
    }

    /**
     * Selects the "Crypto" payment method tab.
     */
    public PaymentPage selectCrypto() {
        wait.until(ExpectedConditions.elementToBeClickable(CRYPTO_TAB)).click();
        return this;
    }

    /**
     * Generic selectPaymentMethod — delegates to the appropriate tab.
     *
     * @param method "vnd" | "crypto"
     */
    public PaymentPage selectPaymentMethod(String method) {
        if ("crypto".equalsIgnoreCase(method)) {
            return selectCrypto();
        }
        return selectVndTransfer();
    }

    /**
     * Initiates payment.
     * For the VND/QR flow there is no in-page button that completes the payment from
     * the FE side (the user scans externally). For Crypto, clicks "Xác nhận thanh toán".
     *
     * TODO: In a real test environment wire up a payment stub and click the appropriate button.
     */
    public void pay() {
        // Try crypto confirm button first
        By cryptoConfirmBtn = By.xpath(
            "//button[contains(normalize-space(),'Xác nhận thanh toán') " +
            "and contains(@class,'bg-gradient-to-r')]"
        );
        List<WebElement> btns = driver.findElements(cryptoConfirmBtn);
        if (!btns.isEmpty()) {
            wait.until(ExpectedConditions.elementToBeClickable(btns.get(0))).click();
        }
        // For VND/bank-transfer there is no actionable button from the FE perspective.
    }

    /**
     * Returns true if the success banner is visible OR the URL contains /booking-success.
     */
    public boolean isSuccess() {
        try {
            WebDriverWait shortWait = new WebDriverWait(driver, Duration.ofSeconds(8));
            shortWait.until(driver -> {
                String url = driver.getCurrentUrl();
                List<WebElement> banners = driver.findElements(SUCCESS_BANNER);
                return (url != null && url.contains("booking-success")) ||
                       banners.stream().anyMatch(WebElement::isDisplayed);
            });
            String url = driver.getCurrentUrl();
            return (url != null && url.contains("booking-success")) ||
                   driver.findElements(SUCCESS_BANNER).stream().anyMatch(WebElement::isDisplayed);
        } catch (Exception e) {
            return false;
        }
    }

    /**
     * Returns the booking ID text shown in the info section.
     */
    public String getDisplayedBookingId() {
        By bookingIdCell = By.xpath(
            "//p[contains(normalize-space(),'Mã booking')]/following-sibling::p"
        );
        try {
            return wait.until(ExpectedConditions.visibilityOfElementLocated(bookingIdCell))
                       .getText().trim();
        } catch (Exception e) {
            return "";
        }
    }
}
