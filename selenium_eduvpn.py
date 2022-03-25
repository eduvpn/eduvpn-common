import sys
from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from pyvirtualdisplay import Display

def login_oauth(driver, authURL):
    driver.get(authURL)
    assert "VPN Portal - Sign In" in driver.title

    portal_user = os.getenv("PORTAL_USER")
    if portal_user is None:
        print("Error: No portal username set, set the PORTAL_USER env var")
        sys.exit(1)

    portal_pass = os.getenv("PORTAL_PASS")
    if portal_pass is None:
        print("Error: No portal password set, set the PORTAL_PASS env var")
        sys.exit(1)

    elem = driver.find_element_by_name("userName")
    elem.clear()
    elem.send_keys(portal_user)

    elem = driver.find_element_by_name("userPass")
    elem.clear()
    elem.send_keys(portal_pass)
    driver.find_element_by_css_selector('.frm > fieldset:nth-child(2) > button:nth-child(2)').click()
    assert "VPN Portal - Approve Application" in driver.title
    driver.find_element_by_css_selector('.frm > fieldset:nth-child(1) > button:nth-child(1)').click()

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Error: no auth url specified")
        sys.exit(1)
    disp = Display()
    disp.start()
    driver = webdriver.Firefox()
    authURL = sys.argv[1]
    login_oauth(driver, authURL)
    driver.close()
    disp.stop()
