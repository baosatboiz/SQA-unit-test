package cinema.pages;

import org.openqa.selenium.By;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.support.ui.ExpectedConditions;
import org.openqa.selenium.support.ui.WebDriverWait;

import java.time.Duration;
import java.util.List;

/**
 * Page Object for the ShowtimesModal overlay.
 *
 * This modal appears after clicking "Xem lịch chiếu" on a movie card (no separate route).
 *
 * Real FE structure (ShowtimesModal.jsx):
 *   Overlay  : <div class="fixed inset-0 bg-black/80 ...">
 *   Inner box: <div class="bg-gray-900 rounded-lg max-w-2xl ...">
 *   Close btn: <button type="button">×</button>  (absolute top-4 right-4)
 *   Showtime buttons (grid): <button type="button" class="p-3 rounded-lg ...">
 *       Contains time text + "Phòng X" + format label
 *   Confirm btn: <button type="button">Đặt vé</button>
 *   Cancel  btn: <button type="button">Đóng</button>
 *
 *   After clicking "Đặt vé" with a selected showtime → navigate("/showtimes/{id}/booking")
 */
public class ShowtimesModalPage {

    private final WebDriver driver;
    private final WebDriverWait wait;

    // Overlay container
    private static final By MODAL_OVERLAY = By.cssSelector("div.fixed.inset-0");

    // Inner modal panel (bg-gray-900)
    private static final By MODAL_PANEL = By.cssSelector("div.fixed.inset-0 div.bg-gray-900");

    /**
     * Showtime choice buttons inside the modal.
     * These are the time-slot buttons in the date-grouped grid.
     * They all have class "p-3 rounded-lg" and are type="button".
     * We exclude the "Đóng" and "Đặt vé" action buttons at the bottom by requiring
     * that they contain a time-like text (or simply by selecting buttons inside the grid container).
     *
     * TODO: If the modal has other p-3 rounded-lg buttons, this selector may be too broad.
     *       A more robust alternative is: .//div[contains(@class,'grid')]//button[@type='button']
     */
    private static final By SHOWTIME_BUTTONS = By.xpath(
        "//div[contains(@class,'fixed') and contains(@class,'inset-0')]" +
        "//div[contains(@class,'grid')]//button[@type='button']"
    );

    // "Đặt vé" confirm button
    private static final By CONFIRM_BUTTON = By.xpath(
        "//div[contains(@class,'fixed') and contains(@class,'inset-0')]" +
        "//button[@type='button' and normalize-space()='Đặt vé']"
    );

    // "Đóng" close button
    private static final By CLOSE_BUTTON = By.xpath(
        "//div[contains(@class,'fixed') and contains(@class,'inset-0')]" +
        "//button[@type='button' and normalize-space()='Đóng']"
    );

    public ShowtimesModalPage(WebDriver driver) {
        this.driver = driver;
        this.wait   = new WebDriverWait(driver, Duration.ofSeconds(15));
    }

    /**
     * Returns true if the modal overlay is currently visible.
     */
    public boolean isOpen() {
        try {
            WebDriverWait shortWait = new WebDriverWait(driver, Duration.ofSeconds(5));
            shortWait.until(ExpectedConditions.visibilityOfElementLocated(MODAL_OVERLAY));
            return true;
        } catch (Exception e) {
            return false;
        }
    }

    /**
     * Waits for the modal to open and returns the number of available showtime buttons.
     */
    public int countShowtimes() {
        wait.until(ExpectedConditions.visibilityOfElementLocated(MODAL_OVERLAY));
        List<WebElement> buttons = driver.findElements(SHOWTIME_BUTTONS);
        return buttons.size();
    }

    /**
     * Selects the showtime button at the given zero-based index.
     *
     * @param index 0-based position among the showtime grid buttons
     */
    public ShowtimesModalPage selectShowtime(int index) {
        wait.until(ExpectedConditions.visibilityOfElementLocated(MODAL_OVERLAY));
        List<WebElement> buttons = wait.until(
            ExpectedConditions.presenceOfAllElementsLocatedBy(SHOWTIME_BUTTONS)
        );
        if (index >= buttons.size()) {
            throw new IndexOutOfBoundsException(
                "Showtime index " + index + " out of range (found " + buttons.size() + ")");
        }
        wait.until(ExpectedConditions.elementToBeClickable(buttons.get(index))).click();
        return this;
    }

    /**
     * Clicks the "Đặt vé" button to navigate to the booking/seat-selection page.
     * Requires a showtime to have been selected first (button is disabled otherwise).
     */
    public void confirmBooking() {
        wait.until(ExpectedConditions.elementToBeClickable(CONFIRM_BUTTON)).click();
    }

    /** Closes the modal by clicking "Đóng". */
    public void close() {
        wait.until(ExpectedConditions.elementToBeClickable(CLOSE_BUTTON)).click();
    }
}
