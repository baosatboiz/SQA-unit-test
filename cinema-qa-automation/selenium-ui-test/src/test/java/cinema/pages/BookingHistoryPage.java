package cinema.pages;

import cinema.utils.ConfigReader;
import org.openqa.selenium.By;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedConditions;
import org.openqa.selenium.support.ui.WebDriverWait;

import java.time.Duration;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Page Object for /booking-history (BookingHistoryPage.jsx)
 *
 * Route: /booking-history
 *
 * Real FE structure:
 *   <h1>Lịch sử đặt vé</h1>
 *
 *   Filter buttons:
 *     <button>Tất cả</button>
 *     <button>Đã xác nhận</button>
 *     <button>Đang chờ</button>
 *     <button>Đã hủy</button>
 *
 *   Search input:
 *     <input type="text" placeholder="Tìm kiếm theo tên phim...">
 *
 *   Each booking card:
 *     <div class="bg-gray-900 rounded-xl p-6 border border-gray-800 ...">
 *       <h3 class="text-xl font-bold text-white ...">MOVIE TITLE</h3>
 *       <span class="px-4 py-2 rounded-full ...">STATUS</span>
 *     </div>
 *
 *   Empty state:
 *     <h3>Chưa có đặt vé nào</h3>
 */
public class BookingHistoryPage {

    private final WebDriver driver;
    private final WebDriverWait wait;

    // Page heading
    private static final By PAGE_HEADING = By.xpath(
        "//h1[contains(normalize-space(),'Lịch sử đặt vé')]"
    );

    /**
     * Booking item cards — each booking is wrapped in:
     * <div class="bg-gray-900 rounded-xl p-6 border border-gray-800 ...">
     */
    private static final By BOOKING_CARDS = By.xpath(
        "//div[contains(@class,'bg-gray-900') and contains(@class,'rounded-xl') " +
        "and contains(@class,'border-gray-800')]"
    );

    // Movie title inside each card
    private static final By CARD_MOVIE_TITLE = By.cssSelector("h3.text-xl.font-bold.text-white");

    // Search input
    private static final By SEARCH_INPUT = By.cssSelector(
        "input[type='text'][placeholder*='Tìm kiếm']"
    );

    // Status filter buttons
    private static final By FILTER_ALL       = By.xpath("//button[normalize-space()='Tất cả']");
    private static final By FILTER_CONFIRMED = By.xpath("//button[normalize-space()='Đã xác nhận']");
    private static final By FILTER_PENDING   = By.xpath("//button[normalize-space()='Đang chờ']");
    private static final By FILTER_CANCELLED = By.xpath("//button[normalize-space()='Đã hủy']");

    // Loading spinner
    private static final By LOADING_SPINNER = By.cssSelector("div.animate-spin");

    // Empty state
    private static final By EMPTY_STATE = By.xpath(
        "//h3[contains(normalize-space(),'Chưa có đặt vé nào')]"
    );

    public BookingHistoryPage(WebDriver driver) {
        this.driver = driver;
        this.wait   = new WebDriverWait(driver, Duration.ofSeconds(20));
    }

    /** Navigate to /booking-history and wait for the page to load. */
    public BookingHistoryPage open() {
        String base = ConfigReader.get("base.url");
        driver.get(base + "/booking-history");
        waitForLoad();
        return this;
    }

    /**
     * Waits for spinner to disappear and the heading to be visible.
     */
    public BookingHistoryPage waitForLoad() {
        try {
            WebDriverWait spinWait = new WebDriverWait(driver, Duration.ofSeconds(5));
            spinWait.until(ExpectedConditions.visibilityOfElementLocated(LOADING_SPINNER));
            wait.until(ExpectedConditions.invisibilityOfElementLocated(LOADING_SPINNER));
        } catch (Exception ignored) {
            // No spinner or already gone
        }
        wait.until(ExpectedConditions.visibilityOfElementLocated(PAGE_HEADING));
        return this;
    }

    /**
     * Returns the number of booking item cards currently displayed.
     * Returns 0 if the empty-state message is shown.
     */
    public int count() {
        // Detect empty state first
        if (!driver.findElements(EMPTY_STATE).isEmpty()) {
            return 0;
        }
        List<WebElement> cards = driver.findElements(BOOKING_CARDS);
        return cards.size();
    }

    /**
     * Returns a list of movie titles from all visible booking cards.
     */
    public List<String> getMovieTitles() {
        List<WebElement> titles = driver.findElements(CARD_MOVIE_TITLE);
        return titles.stream()
                     .map(el -> el.getText().trim())
                     .filter(t -> !t.isEmpty())
                     .collect(Collectors.toList());
    }

    /**
     * Types in the search box to filter bookings by movie title (client-side filter).
     *
     * @param keyword partial or full movie title
     */
    public BookingHistoryPage search(String keyword) {
        WebElement input = wait.until(
            ExpectedConditions.visibilityOfElementLocated(SEARCH_INPUT)
        );
        input.clear();
        input.sendKeys(keyword);
        return this;
    }

    /**
     * Clicks a status filter button.
     *
     * @param status "all" | "confirmed" | "pending" | "cancelled"  (case-insensitive)
     */
    public BookingHistoryPage filterByStatus(String status) {
        By tab;
        switch (status.toLowerCase()) {
            case "confirmed":
            case "da_xac_nhan":
                tab = FILTER_CONFIRMED;
                break;
            case "pending":
            case "dang_cho":
                tab = FILTER_PENDING;
                break;
            case "cancelled":
            case "da_huy":
                tab = FILTER_CANCELLED;
                break;
            default:
                tab = FILTER_ALL;
        }
        wait.until(ExpectedConditions.elementToBeClickable(tab)).click();
        // Brief wait for server re-fetch (useEffect triggers on statusFilter change)
        try {
            WebDriverWait spinWait = new WebDriverWait(driver, Duration.ofSeconds(3));
            spinWait.until(ExpectedConditions.visibilityOfElementLocated(LOADING_SPINNER));
            wait.until(ExpectedConditions.invisibilityOfElementLocated(LOADING_SPINNER));
        } catch (Exception ignored) {
            // fast response or no spinner
        }
        return this;
    }

    /**
     * Returns true if the "Chưa có đặt vé nào" empty-state message is visible.
     */
    public boolean isEmpty() {
        return !driver.findElements(EMPTY_STATE).isEmpty() &&
               driver.findElements(EMPTY_STATE).get(0).isDisplayed();
    }
}
