# Automated Rail Ticket Booker And Notifier

---
## How to Use the Program
Just download the automated-ticket-booker.exe file 
from https://github.com/minhaz725/Automated-Rail-Ticket-Booker-And-Notifier
and run it. Your operating system might block it for the first time just click on more info and run it anyway. That's it done!
(If it still blocks, open a terminal in the downloaded folder and open it from there.)

**Basic things you should know before running the program**:

- Google Chrome must be installed in your Operating System (Windows / Mac / Linux).


- You need to log in to the ticket site(https://eticket.railway.gov.bd/), if you're already logged in, skip.


- There's no input validation currently, so check spellings carefully.


- Know the Seat Types: 
  - S_CHAIR-> Non AC Chair
  - SNIGDHA-> AC Chair
  - F_BERTH-> Non AC Bed (Cabin)
  - AC_B-> AC Bed (Cabin)
  - F_SEAT-> Non AC Chair (Cabin)
  - AC_S-> AC Chair (Cabin)
  - SHOVAN-> 2nd class Bench
  

- Write only one train name. If you want to search multiple trains together, then run the app again and write another.


- The seat face option has "Towards Dhaka" and "From Dhaka" options. By choosing "Towards Dhaka", the app tries to find seats from seat 1. By choosing the opposite option, the app tries to find seats from the last seat. Usually, if the train goes to Dhaka, seats from 1 to half (e.g., 1-30) are forward-facing. If the train goes out from Dhaka, seats from the last to half (e.g., 60-30) are forward-facing. For other destinations, you need to know the train face info and select options accordingly.


- Cross out the "Go to booking page" checkbox if you want to review seats before booking.


- If you plan to purchase tickets at the ticket release day (10 days prior) during Eid time:
    - Start the app at 2:00 PM if you want to travel to Purbanchal.
    - Start the app at 8:00 AM if you want to travel to Poshchimanchal.
    - Start the app at any time if it's between 9 to 1 day before the journey.
    - If you search once, your previous search will always be saved.
    - If you see zero tickets, don't worry. Just keep it running (internet must be on), the app will notify you.
    - OTP might not come timely, and the seat might not be clickable. Here, just run the program again and again.
    - The program will always automatically detect empty seats and book them, so don't lose hope.

---
**Disclaimer**: If the website gets major changes, the program won't work. I'll try to update it as soon as possible. If you face any issues or bugs, please let me know.

## How to Run the Code

- This app requires go 1.20.6, make sure it's installed otherwise download from here: https://go.dev/dl/
- Run the following commands and make sure they are working:
  - Go Root:
    ```bash
    go env GOROOT
  - Go Version:
    ```bash
    go version
- Install Fyne following the guide from official doc: https://docs.fyne.io/started/
- Clone the repository and open the terminal inside the directory.
- Run the following commands to download and setup dependencies:
  - Tidy:
    ```bash
    go mod tidy
  - Download:
    ```bash
    go mod download
  - Vendor:
    ```bash
    go mod vendor
- Go inside cmd/main and run build command, which will take several minutes for the first time depending on your machine's power:
    ```bash
    go build
- Finally, run the program:
    ```bash
    go run main.go 
  
## Create Portable Executables (.exe files)
- Just run create-exe.bat file (windows) or create.exe.sh (linux/mac)