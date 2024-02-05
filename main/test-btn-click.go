package main

//
//import (
//	"context"
//	"github.com/chromedp/chromedp"
//	"log"
//	"time"
//)
//
//func main() {
//	// create chrome instance
//
//	opts := append(chromedp.DefaultExecAllocatorOptions[:],
//		chromedp.Flag("headless", false),
//	)
//	// Create a context with options.
//	initialCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
//	ctx, cancel := chromedp.NewContext(
//		initialCtx,
//		chromedp.WithDebugf(log.Printf),
//	)
//	defer cancel()
//
//	// create a timeout
//	ctx, cancel = context.WithTimeout(ctx, 300*time.Second)
//	defer cancel()
//
//	// navigate to a page, wait for an element, click
//	var example string
//	err := chromedp.Run(ctx,
//		chromedp.Navigate(`https://eticket.railway.gov.bd/booking/train/search?fromcity=Dhaka&tocity=Cox%27s%20Bazar&doj=16-Feb-2024&class=S_CHAIR`),
//
//		// wait for footer element is visible (ie, page is loaded)
//		//chromedp.WaitVisible(`button.modify_search.mod_search`),
//		chromedp.WaitReady(".seat-availability-box"),
//		// find and click "Example" link
//		//chromedp.Click(`div.button_set > button:not(.button_mr)`, chromedp.NodeVisible),
//		chromedp.Sleep(3*time.Second),
//		chromedp.Click(`button.modify_search.mod_search`, chromedp.NodeVisible),
//		//chromedp.Click(`app-single-trip.ng-star-inserted h2:contains("SONAR BANGLA EXPRESS (788)") + button.trip-details-btn`, chromedp.NodeVisible),
//		//chromedp.Click(`//app-single-trip//button[contains(@class, 'trip-details-btn')]`, chromedp.NodeVisible),
//		chromedp.Evaluate(`(() => {
//			const headers = Array.from(document.querySelectorAll('h2'));
//			const header = headers.find(h => h.innerText === 'SONAR BANGLA EXPRESS (788)');
//			if (!header) throw new Error('Header not found');
//			const btn = header.closest('app-single-trip').querySelector('.trip-details-btn');
//			if (!btn) throw new Error('Button not found');
//			btn.click();
//		})()`, &example),
//
//		//chromedp.Click(`span.seat-class-name:contains("SNIGDHA") + .seat-availability-box .book-now-btn`, chromedp.NodeVisible),
//
//		// retrieve the text of the textarea
//		//chromedp.Value(`#example-After textarea`, &example),
//		chromedp.Sleep(10*time.Second),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//	log.Printf("Go's time.After example:\n%s", example)
//}
