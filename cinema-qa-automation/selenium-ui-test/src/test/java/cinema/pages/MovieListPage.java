package cinema.pages;

import cinema.utils.ConfigReader;
import org.openqa.selenium.By;
import org.openqa.selenium.WebDriver;
import org.openqa.selenium.WebElement;
import org.openqa.selenium.interactions.Actions;
import org.openqa.selenium.support.ui.ExpectedConditions;
import org.openqa.selenium.support.ui.WebDriverWait;

import java.time.Duration;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Page Object for /movies
 *
 * Real FE structure (MoviesPage.jsx + MovieListItem.jsx):
 *
 * Movie cards:
 *   <div class="group bg-gray-800/50 backdrop-blur-sm rounded-lg ...">
 *     <h3 class="text-white font-bold ...">TITLE</h3>
 *     <!-- hover-only button, visible after hover: -->
 *     <button type="button" class="... opacity-0 group-hover:opacity-100 ...">Xem lịch chiếu</button>
 *   </div>
 *
 * Filter tabs (MovieFilterTabs component — status filter buttons):
 *   Buttons containing text "Tất cả" | "Đang chiếu" | "Sắp chiếu"
 *
 * NOTE: MoviesPage has no free-text search input. Filtering is done via status tabs.
 */
public class MovieListPage {

    private final WebDriver driver;
    private final WebDriverWait wait;
    private final Actions actions;

    // Movie card container — tolerant selector using XPath with contains
    private static final By MOVIE_CARDS = By.xpath(
        "//div[contains(@class,'group') and contains(@class,'bg-gray-800')]"
    );

    // Title inside a card
    private static final By CARD_TITLE = By.cssSelector("h3.text-white.font-bold, h3.font-bold");

    // "Xem lịch chiếu" button inside a card (opacity-0 by default, visible on hover)
    private static final By VIEW_SHOWTIMES_BUTTON = By.xpath(
        ".//button[@type='button' and contains(text(),'Xem lịch chiếu')]"
    );

    // Status filter tab buttons
    private static final By FILTER_TAB_ALL      = By.xpath("//button[normalize-space()='Tất cả']");
    private static final By FILTER_TAB_SHOWING  = By.xpath("//button[normalize-space()='Đang chiếu']");
    private static final By FILTER_TAB_UPCOMING = By.xpath("//button[normalize-space()='Sắp chiếu']");

    public MovieListPage(WebDriver driver) {
        this.driver  = driver;
        this.wait    = new WebDriverWait(driver, Duration.ofSeconds(15));
        this.actions = new Actions(driver);
    }

    /** Navigate to /movies and wait for at least one card to appear. */
    public MovieListPage open() {
        String base = ConfigReader.get("base.url");
        driver.get(base + "/movies");
        wait.until(ExpectedConditions.presenceOfElementLocated(MOVIE_CARDS));
        return this;
    }

    /**
     * Filter movies by status using the tab buttons.
     *
     * @param status "all" | "showing" | "upcoming"  (case-insensitive)
     */
    public MovieListPage filter(String status) {
        By tabLocator;
        switch (status.toLowerCase()) {
            case "showing":
            case "dang_chieu":
                tabLocator = FILTER_TAB_SHOWING;
                break;
            case "upcoming":
            case "sap_chieu":
                tabLocator = FILTER_TAB_UPCOMING;
                break;
            default:
                tabLocator = FILTER_TAB_ALL;
        }
        wait.until(ExpectedConditions.elementToBeClickable(tabLocator)).click();
        // Brief wait for the list to re-render
        wait.until(ExpectedConditions.presenceOfElementLocated(MOVIE_CARDS));
        return this;
    }

    /** Returns the number of visible movie cards. */
    public int countResults() {
        List<WebElement> cards = driver.findElements(MOVIE_CARDS);
        return cards.size();
    }

    /** Returns all visible movie titles. */
    public List<String> getTitles() {
        List<WebElement> titles = wait.until(
            ExpectedConditions.presenceOfAllElementsLocatedBy(CARD_TITLE)
        );
        return titles.stream()
                     .map(el -> el.getText().trim())
                     .filter(t -> !t.isEmpty())
                     .collect(Collectors.toList());
    }

    /**
     * Hover over the first movie card and click "Xem lịch chiếu".
     * The button is only visible after hover (opacity-0 → opacity-100 via Tailwind group-hover).
     */
    public void clickFirst() {
        List<WebElement> cards = wait.until(
            ExpectedConditions.presenceOfAllElementsLocatedBy(MOVIE_CARDS)
        );
        if (cards.isEmpty()) {
            throw new RuntimeException("No movie cards found on /movies page");
        }
        WebElement firstCard = cards.get(0);
        // Hover to reveal the button
        actions.moveToElement(firstCard).perform();

        // Wait for the button to become clickable (CSS transition ~300 ms)
        WebElement btn = wait.until(
            ExpectedConditions.elementToBeClickable(firstCard.findElement(VIEW_SHOWTIMES_BUTTON))
        );
        btn.click();
    }

    /**
     * Hover over the card at the given zero-based index and click "Xem lịch chiếu".
     */
    public void clickCardAt(int index) {
        List<WebElement> cards = wait.until(
            ExpectedConditions.presenceOfAllElementsLocatedBy(MOVIE_CARDS)
        );
        if (index >= cards.size()) {
            throw new IndexOutOfBoundsException(
                "Card index " + index + " out of range (found " + cards.size() + " cards)");
        }
        WebElement card = cards.get(index);
        actions.moveToElement(card).perform();
        WebElement btn = wait.until(
            ExpectedConditions.elementToBeClickable(card.findElement(VIEW_SHOWTIMES_BUTTON))
        );
        btn.click();
    }
}
