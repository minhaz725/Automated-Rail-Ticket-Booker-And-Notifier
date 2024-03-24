(() => {
    const headers = Array.from(document.querySelectorAll("h2"));
    const header = headers.find((h) =>
        h.innerText.includes("` + selectedSpecificTrain + `")
    );
    if (!header) throw new Error("Header not found");
    const appSingleTrip = header.closest("app-single-trip");
    if (!appSingleTrip) throw new Error("Parent component not found");

    // Filter single-seat-class divs by the text content of the seat-class-name span
    const seatClassDivs = Array.from(
        appSingleTrip.querySelectorAll(".single-seat-class")
    );

    let seatTypeArrayLength = parseInt(
        "` + strconv.Itoa(len(arguments.SEAT_TYPE_ARRAY)) + `"
    );
    let bookNowBtn;

    for (let i = 0; i < seatTypeArrayLength; i++) {
        let seatType;
        if (i == 0) seatType = "` + arguments.SEAT_TYPE_ARRAY[0] + `";
        if (i == 1) seatType = "` + arguments.SEAT_TYPE_ARRAY[1] + `";
        //if(i==2) seatType = '` + /*constants.SEAT_TYPE_ARRAY[2] +*/ `'
        let seatDiv = seatClassDivs.find((div) => {
            let seatNameSpan = div.querySelector(".seat-class-name");
            return seatNameSpan && seatNameSpan.innerText.trim() === seatType;
        });
        //throw new Error('Seat class div not found');

        // Find and click the book now button within the specific seat class div
        bookNowBtn = seatDiv.querySelector(".book-now-btn-wrapper .book-now-btn");
        if (bookNowBtn != null) break;
    }

    if (!bookNowBtn)
        throw new Error("Book now button not found for All given Types" + seatType);

    bookNowBtn.click();

    const waitForSelectBogie = new Promise((resolve, reject) => {
        setTimeout(() => {
            const bogieSelection = document.getElementById("select-bogie");
            if (!bogieSelection)
                reject(new Error("Bogie selection dropdown not found"));

            const extractNumber = (text) => {
                const match = text.match(/\d+/);
                return match ? parseInt(match[0]) : 0;
            };

            const options = Array.from(bogieSelection.options);
            const highestOption = options.reduce((highest, current) => {
                const highestNumber = extractNumber(highest.text);
                const currentNumber = extractNumber(current.text);
                return currentNumber > highestNumber ? current : highest;
            }, options[0]);

            const coachWithHighestSeat = highestOption.text.split(" - ")[0];

            const coachOption = Array.from(bogieSelection.options).find((option) =>
                option.text.includes(coachWithHighestSeat)
            );

            bogieSelection.value = coachOption.value;
            bogieSelection.dispatchEvent(new Event("change", { bubbles: true }));

            resolve(coachWithHighestSeat);
        }, 1000); // Delay of 1000 milliseconds (1 second)
    });

    const clickSeatButtons = (coachWithHighestSeat) => {
        return new Promise((resolve, reject) => {
            setTimeout(() => {
                const clickSeatButton = (seatNumber) => {
                    const selector =
                        '.btn-seat.seat-available[title^="' +
                        coachWithHighestSeat +
                        '-"][title$="-' +
                        seatNumber +
                        '"]';
                    const seatButton = document.querySelector(selector);

                    if (seatButton) {
                        seatButton.click();
                        return true; // Seat button found and clicked
                    }
                    return false; // Seat button not found
                };

                let seatNumber = 1;
                let seatCount = parseInt(
                    "` + strconv.Itoa(int(arguments.SEAT_COUNT)) + `"
                );

                // Loop to find and click on seat buttons
                while (seatCount > 0) {
                    if (clickSeatButton(seatNumber)) {
                        seatCount--;
                    }
                    seatNumber++; // Increment the seat number for the next iteration
                }

                resolve(); // Resolve the promise after clicking on seats
            }, 500); // Delay of 500 milliseconds
        });
    };

    waitForSelectBogie
        .then((coachWithHighestSeat) => {
            clickSeatButtons(coachWithHighestSeat)
                .then(() => {
                    // After clicking on seats, find and click the "Continue Purchase" button
                    let purchasePage = parseInt("` + strconv.Itoa(int(arguments.GO_TO_BOOK_PAGE)) + `");
                    if(purchasePage == 1) {
                        setTimeout(() => {
                            const continueButton = document.querySelector(".continue-btn");
                            if (!continueButton)
                                throw new Error("Continue Purchase button not found");
                            continueButton.click();
                        }, 500); // Delay of 500 milliseconds after clicking on seats
                    }
                })
                .catch((error) => {
                    console.error(error); // Handle any errors from clicking on seat buttons
                });
            a;
        })
        .catch((error) => {
            console.error(error); // Handle any errors from selecting the bogie
        });

    return true;
})();