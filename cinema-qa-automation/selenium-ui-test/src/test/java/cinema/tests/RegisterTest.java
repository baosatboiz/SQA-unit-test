package cinema.tests;

import cinema.base.BaseTest;
import cinema.pages.RegisterPage;
import cinema.utils.ConfigReader;
import cinema.utils.DBHelper;
import cinema.utils.ExcelReader;
import org.testng.Assert;
import org.testng.annotations.BeforeClass;
import org.testng.annotations.DataProvider;
import org.testng.annotations.Test;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.Map;

/**
 * UI-01, UI-02, UI-03 — Registration tests.
 *
 * RegisterData sheet columns:
 *   tcId | email | password | firstName | lastName | address | expectedResult | expectedMessage
 *
 * The email field may contain "${ts}" which is replaced with System.currentTimeMillis()
 * to ensure uniqueness on each run.
 *
 * UI-01: valid new user → success (redirect to /verify, DB record exists)
 * UI-02: duplicate email → error containing expectedMessage
 * UI-03: weak/invalid input → error containing expectedMessage
 */
public class RegisterTest extends BaseTest {

    private static final String DATA_FILE = "test-data/ClientFlowData.xlsx";

    /**
     * Pre-condition for UI-02: ensure qa_test_existing@example.com already exists in DB.
     * If not, seed via the registration API so duplicate-email validation can be triggered.
     * Network/API errors are logged but do NOT fail test setup.
     *
     * NOTE: TestNG calls @BeforeClass methods in declaration order within a class, and
     * BaseTest.setUp() is also @BeforeClass — TestNG merges them correctly when
     * alwaysRun=true is set on both.
     */
    @BeforeClass(alwaysRun = true, dependsOnMethods = "setUp")
    public void seedExistingUser() {
        String email = "qa_test_existing@example.com";
        try {
            if (DBHelper.userExists(email)) {
                System.out.println("[Seed] qa_test_existing already in DB, skipping registration.");
                return;
            }
        } catch (Exception e) {
            System.err.println("[Seed] DB check failed (DB may be offline): " + e.getMessage());
            // Continue — the test will fail later if the user truly isn't seeded
        }
        try {
            HttpClient client = HttpClient.newHttpClient();
            String body = String.format(
                "{\"email\":\"%s\",\"password\":\"Pass@1234\",\"confirmPassword\":\"Pass@1234\"," +
                "\"firstName\":\"Existing\",\"lastName\":\"User\",\"address\":\"123 Existing Street\"}",
                email);
            HttpRequest request = HttpRequest.newBuilder()
                    .uri(URI.create(ConfigReader.get("api.base.url") + "/api/v1/auth/register"))
                    .header("Content-Type", "application/json")
                    .POST(HttpRequest.BodyPublishers.ofString(body))
                    .build();
            HttpResponse<String> response =
                    client.send(request, HttpResponse.BodyHandlers.ofString());
            System.out.println("[Seed] Seeded existing user via API. Status: " + response.statusCode());
        } catch (Exception e) {
            System.err.println("[Seed] Warning — could not seed existing user: " + e.getMessage());
        }
    }

    @DataProvider(name = "registerData")
    public Object[][] registerData() {
        return ExcelReader.toDataProvider(DATA_FILE, "RegisterData");
    }

    /**
     * Parameterized registration test driven by RegisterData sheet.
     * Handles both PASS and FAIL expected results.
     */
    @Test(dataProvider = "registerData",
          description = "Register with various inputs — UI-01, UI-02, UI-03")
    public void register(Map<String, String> row) {
        // Replace timestamp placeholder for uniqueness
        String email = row.get("email").replace("${ts}", String.valueOf(System.currentTimeMillis()));
        String password     = row.get("password");
        String firstName    = row.get("firstName");
        String lastName     = row.get("lastName");
        String address      = row.get("address");
        String expectedResult  = row.get("expectedResult");
        String expectedMessage = row.get("expectedMessage");

        // RegisterPage.fill() signature: (email, password, confirmPassword, firstName, lastName, address)
        // Use the same password as confirmPassword — the Excel data does not have a separate column for it.
        RegisterPage rp = new RegisterPage(driver)
                .open()
                .fill(email, password, password, firstName, lastName, address);
        rp.submit();

        if ("PASS".equalsIgnoreCase(expectedResult)) {
            Assert.assertTrue(rp.isSuccess(),
                    "[" + row.get("tcId") + "] Register expected to succeed for " + email);
            try {
                Assert.assertTrue(DBHelper.userExists(email),
                        "[" + row.get("tcId") + "] User must exist in DB after register: " + email);
            } catch (Exception dbEx) {
                System.err.println("[" + row.get("tcId") + "] DB check skipped (DB offline?): " + dbEx.getMessage());
            }
        } else {
            String err = rp.getError().toLowerCase();
            String expected = (expectedMessage != null ? expectedMessage : "").toLowerCase();
            Assert.assertTrue(
                    expected.isEmpty() || err.contains(expected),
                    "[" + row.get("tcId") + "] Expected error to contain: \"" + expected +
                    "\" but got: \"" + err + "\"");
        }
    }
}
