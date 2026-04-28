package cinema.tests;

import cinema.base.BaseTest;
import cinema.pages.MovieListPage;
import cinema.pages.ShowtimesModalPage;
import cinema.utils.ExcelReader;
import org.testng.Assert;
import org.testng.annotations.DataProvider;
import org.testng.annotations.Test;

import java.util.Map;

/**
 * UI-06, UI-07 — Movie Browse tests.
 *
 * NOTE: MovieListPage has NO free-text search input.
 * Filtering is done via status tabs ("Tất cả" / "Đang chiếu" / "Sắp chiếu").
 * The SearchData sheet's "keyword" column is therefore used as a status filter
 * value ("all", "showing", or "upcoming").  If the keyword does not match any
 * filter tab it defaults to "all".
 *
 * SearchData sheet columns:
 *   tcId | keyword | expectedMinResults
 *
 * UI-06: Open /movies, apply filter by keyword (status tab), assert countResults() >= expectedMinResults.
 * UI-07: Open /movies, click first movie card, assert ShowtimesModalPage opens
 *        and has at least 1 showtime.
 */
public class MovieBrowseTest extends BaseTest {

    private static final String DATA_FILE = "test-data/ClientFlowData.xlsx";

    @DataProvider(name = "searchData")
    public Object[][] searchData() {
        return ExcelReader.toDataProvider(DATA_FILE, "SearchData");
    }

    /**
     * UI-06: Filter movie list and verify the minimum number of results.
     *
     * MovieListPage does not have a search(keyword) method.
     * We map the keyword to filter(status):
     *   "all" | "" | null   → filter("all")
     *   "showing"           → filter("showing")
     *   "upcoming"          → filter("upcoming")
     * Any other keyword is treated as "all".
     */
    @Test(dataProvider = "searchData",
          description = "UI-06: Filter movies list and assert minimum result count")
    public void filterMovies_UI06(Map<String, String> row) {
        String keyword         = row.get("keyword");
        int expectedMinResults = parseMinResults(row.get("expectedMinResults"));

        MovieListPage mlp = new MovieListPage(driver).open();

        // Map keyword to status-filter tab value
        String filterValue = resolveFilterValue(keyword);
        mlp.filter(filterValue);

        int actualCount = mlp.countResults();
        Assert.assertTrue(actualCount >= expectedMinResults,
                "[" + row.get("tcId") + "] Expected at least " + expectedMinResults +
                " movie result(s) for filter \"" + filterValue + "\" but found " + actualCount);
    }

    /**
     * UI-07: Click the first movie card, verify the showtimes modal opens,
     * and that at least one showtime slot is available.
     */
    @Test(description = "UI-07: Click first movie card, assert ShowtimesModal opens with >= 1 showtime")
    public void clickFirstMovieShowtimes_UI07() {
        MovieListPage mlp = new MovieListPage(driver).open();

        int cardCount = mlp.countResults();
        Assert.assertTrue(cardCount >= 1,
                "UI-07: Expected at least 1 movie card on /movies but found " + cardCount);

        // Hover + click first card's "Xem lịch chiếu" button
        mlp.clickFirst();

        ShowtimesModalPage modal = new ShowtimesModalPage(driver);

        Assert.assertTrue(modal.isOpen(),
                "UI-07: ShowtimesModal should open after clicking first movie card");

        int showtimeCount = modal.countShowtimes();
        Assert.assertTrue(showtimeCount >= 1,
                "UI-07: ShowtimesModal should display at least 1 showtime, but found " + showtimeCount);
    }

    // ── helpers ──────────────────────────────────────────────────────────────

    /**
     * Maps an Excel keyword value to the MovieListPage.filter() status string.
     * Accepted inputs: "all", "showing", "upcoming" (case-insensitive).
     * Empty / unrecognised values default to "all".
     */
    private String resolveFilterValue(String keyword) {
        if (keyword == null || keyword.trim().isEmpty()) return "all";
        switch (keyword.trim().toLowerCase()) {
            case "showing":
            case "dang_chieu":
            case "dang chieu":
                return "showing";
            case "upcoming":
            case "sap_chieu":
            case "sap chieu":
                return "upcoming";
            default:
                return "all";
        }
    }

    /** Safely parse expectedMinResults; returns 0 on parse error. */
    private int parseMinResults(String value) {
        try {
            return Integer.parseInt(value == null ? "0" : value.trim());
        } catch (NumberFormatException e) {
            return 0;
        }
    }
}
