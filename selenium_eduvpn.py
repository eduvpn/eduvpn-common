import sys
from selenium import webdriver
from selenium.webdriver.common.keys import Keys
from pyvirtualdisplay import Display

def login_oauth(driver, authURL):
    driver.get(authURL)
    assert "VPN Portal - Sign In" in driver.title
    elem = driver.find_element_by_name("userName")
    elem.clear()
    elem.send_keys("docker")

    elem = driver.find_element_by_name("userPass")
    elem.clear()
    elem.send_keys("docker")
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
