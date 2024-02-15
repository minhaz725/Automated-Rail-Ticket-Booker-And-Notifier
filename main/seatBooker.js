(() => {
    const headers = Array.from(document.querySelectorAll('h2'));
    const header = headers.find(h => h.innerText.includes(constants.SPECIFIC_TRAIN));
    if (!header) throw new Error('Header not found');
    const appSingleTrip = header.closest('app-single-trip');
    if (!appSingleTrip) throw new Error('Parent component not found');

    // Filter single-seat-class divs by the text content of the seat-class-name span
    const seatClassDivs = Array.from(appSingleTrip.querySelectorAll('.single-seat-class'));
    const seatDiv = seatClassDivs.find(div => {
        const seatNameSpan = div.querySelector('.seat-class-name');
        return seatNameSpan && seatNameSpan.innerText.trim() === '`+constants.SEAT_TYPE+`';
    });
    if (!seatDiv) throw new Error('Seat class div not found');

    // Find and click the book now button within the specific seat class div
    const bookNowBtn = seatDiv.querySelector('.book-now-btn-wrapper .book-now-btn');
    if (!bookNowBtn) throw new Error('Book now button not found');
    bookNowBtn.click();

    setTimeout(() => {
        // Find the select element
        const bogieSelection = document.getElementById('select-bogie');
        if (!bogieSelection) throw new Error('Bogie selection dropdown not found');

        // Find the option that contains the coach numb
        const coachOption = Array.from(bogieSelection.options).find(option => option.text.includes('`+constants.COACH_NUMB+`'));
        if (!coachOption) throw new Error('Option with text `+constants.COACH_NUMB+` not found');

        // Set the selected option to the one found
        bogieSelection.value = coachOption.value;
        // Dispatch an input event to simulate user interaction
        bogieSelection.dispatchEvent(new Event('change', { bubbles: true }));

        setTimeout(() => {

            const seatOne = document.querySelector('.btn-seat.seat-available[title="`+constants.COACH_NUMB+constants.SEAT_ONE_NUMB+`"]');
            if (!seatOne) throw new Error('seatOne button not found');
            seatOne.click();
            const seatTwo = document.querySelector('.btn-seat.seat-available[title="`+constants.COACH_NUMB+constants.SEAT_TWO_NUMB+`"]');
            if (!seatTwo) throw new Error('seatTwo button not found');
            seatTwo.click();
            // setTimeout(() => {
            //     const continueButton = document.querySelector('.continue-btn');
            //     if (!continueButton) throw new Error('Continue Purchase button not found');
            //     continueButton.click();
            // },  100);
        },  100);
    },  100);
})();