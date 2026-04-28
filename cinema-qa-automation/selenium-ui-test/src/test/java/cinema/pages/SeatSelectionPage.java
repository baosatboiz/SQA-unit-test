package cinema.pages;

import org.openqa.selenium.By;
import org.openqa.selenium.JavascriptExecutor;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedConditions;
import org.openqa.selenium.support.ui.WebDriverWait;

import java.time.Duration;
import java.util.List;

/**
 * Page Object for /showtimes/:id/booking (BookingPage.jsx + SeatGrid.jsx)
 *
 * Seat color logic (client colorScheme in SeatGrid.jsx):
 *   Available  (REGULAR)  → bg-green-600 border-green-500  cursor-pointer
 *   Available  (VIP)      → bg-yellow-600 border-yellow-500 cursor-pointer
 *   Available  (COUPLE)   → bg-pink-600   border-pink-500   cursor-pointer
 *   Selected              → bg-red-600    border-red-500
 *   Booked / OCCUPIED     → bg-gray-500   border-gray-400   cursor-not-allowed
 *   Locked (temp)         → bg-orange-600 border-orange-500 cursor-not-allowed
 *
 * Seat element: <button class="w-8 h-8 border-2 rounded text-xs font-semibold ...">NN</button>
 *   where NN is zero-padded seat number (01, 02, …)
 *
 * "Tiếp tục thanh toán" button (BookingPage.jsx summary sidebar):
 *   <button class="w-full mt-4 bg-red-600 ...">Tiếp tục thanh toán</button>
 */
public class SeatSelectionPage {

    private final WebDriver driver;
    private final WebDriverWait wait;

    /**
     * All seat buttons rendered by SeatGrid.
     * SeatGrid renders: <button class="w-8 h-8 border-2 rounded text-xs font-semibold ...">
     * Couple seats use w-12 instead of w-8.
     * We select all buttons that match at least border-2 + rounded + text-xs.
     */
    private static final By ALL_SEAT_BUTTONS = By.cssSelector(
        "button.border-2.rounded.text-xs.font-semibold"
    );

    /**
     * Available seat buttons: those containing bg-green-600, bg-yellow-600, or bg-pink-600.
     * XPath "or" on @class is more reliable than CSS multi-class for Tailwind.
     *
     * TODO: If Tailwind purges produce different class names in production build, update here.
     */
    private static final By AVAILABLE_SEAT_BUTTONS = By.xpath(
        "//button[contains(@class,'bg-green-600') or contains(@class,'bg-yellow-600') " +
        "or contains(@class,'bg-pink-600')]" +
        "[contains(@class,'border-2') and contains(@class,'rounded')]"
    );

    /**
     * Booked/occupied seat buttons: bg-gray-500 cursor-not-allowed.
     */
    private static final By BOOKED_SEAT_BUTTONS = By.xpath(
        "//button[contains(@class,'bg-gray-500') and contains(@class,'cursor-not-allowed')]" +
        "[contains(@class,'border-2') and contains(@class,'rounded')]"
    );

    /**
     * "Tiếp tục thanh toán" button in the summary sidebar (appears when ≥1 seat selected).
     * Fallback XPath also matches partial text for robustness.
     */
    private static final By PROCEED_BUTTON = By.xpath(
        "//button[contains(normalize-space(),'Tiếp tục') or " +
        "contains(normalize-space(),'Thanh toán')]" +
        "[contains(@class,'bg-red-600')]"
    );

    public SeatSelectionPage(WebDriver driver) {
        this.driver = driver;
        this.wait   = new WebDriverWait(driver, Duration.ofSeconds(20));
    }

    /**
     * Waits for the seat grid to load, then clicks the first available seat.
     * Returns the seat element's title attribute (e.g. "A01 - Ghế thường").
     *
     * @throws RuntimeException if no available seats are found
     */
    public String selectFirstAvailable() {
        // Wait until at least one seat button is present
        wait.until(ExpectedConditions.presenceOfElementLocated(ALL_SEAT_BUTTONS));

        List<WebElement> availableSeats = wait.until(
            ExpectedConditions.presenceOfAllElementsLocatedBy(AVAILABLE_SEAT_BUTTONS)
        );
        if (availableSeats.isEmpty()) {
            throw new RuntimeException("No available seats found on seat selection page");
        }
        WebElement seat = availableSeats.get(0);
        String title = seat.getAttribute("title");
        // Scroll into view before clicking (in case seat grid overflows)
        ((JavascriptExecutor) driver).executeScript("arguments[0].scrollIntoView(true);", seat);
        wait.until(ExpectedConditions.elementToBeClickable(seat)).click();
        return title != null ? title : "";
    }

    /**
     * Returns true if at least one booked seat is rendered with cursor-not-allowed
     * and the HTML disabled attribute or the CSS class indicating non-interactive state.
     */
    public boolean isBookedSeatDisabled() {
        wait.until(ExpectedConditions.presenceOfElementLocated(ALL_SEAT_BUTTONS));
        List<WebElement> bookedSeats = driver.findElements(BOOKED_SEAT_BUTTONS);
        if (bookedSeats.isEmpty()) {
            // No booked seats present — cannot assert, return false
            return false;
        }
        // Verify: cursor-not-allowed class must be present
        WebElement sample = bookedSeats.get(0);
        String cls = sample.getAttribute("class");
        return cls != null && cls.contains("cursor-not-allowed");
    }

    /**
     * Clicks the "Tiếp tục thanh toán" button in the summary sidebar.
     * This button only appears after at least one seat has been selected.
     */
    public void confirm() {
        wait.until(ExpectedConditions.elementToBeClickable(PROCEED_BUTTON)).click();
    }

    /**
     * Returns all currently rendered seat button elements.
     * Useful for counting total seats.
     */
    public int countTotalSeats() {
        wait.until(ExpectedConditions.presenceOfElementLocated(ALL_SEAT_BUTTONS));
        return driver.findElements(ALL_SEAT_BUTTONS).size();
    }
}
