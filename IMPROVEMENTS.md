# BUGS
## Ordering of speakers list does not work as intended.
- FLINTA* items are always on top, but gender-quotation should interleave the two lists, with priority to the FLINTA* list if in doubt. Currently FLINTA* speakers are always on top.
- First speaker quotation should always be first WITHIN their gender quoted lists, so the gender interleaving should always take priority and only when considering which speaker from the respective gender list gets to speak first, you should consider firs speaker quotation
## Voting: submitting wrong amount of choices.
- The voting form allows you to send the wrong amount of choices and has weird error messages all over the screen, while casting your ballot without having it be counted.
- I have not tested it, but check whether this behaviour also applies to the manual submission form in the admin panel.
## Moderate: Adding speakers via search
- The intended behaviour when using the search to type in an exact participant number and hitting enter is to put that exact participant on the speakers list and do nothing if they already are. Currently it seems that if I i.e. type in `3`, it adds the participant with ID 2.
## Help: Color theme does not change image choice.
- AFAIK the help page images should also have dark mode variants, but I always see the light mode variants. The same goes for mobile layout mode. Language does seem to change the images used.
## Moderate: Participant view layout is wrong.
- The toggle switches for "Chair" and "FLINTA*" are below the entire row, instead of being right of the two buttons stacked vertically.
## Moderate: QR Code Button has a malformed SVG icon.
- The QR code does not look right
## Moderate: Past speakers list items do not show their time
- The past speakers list items should show their speaking time.
## Regular login page login card extends to bottom of available space.
- The card should shrink to fit and be centered vertically as well.
## Admin login page has no button to get back to regular login page
- self explanatory, that back button should exist
## Admin login page looks different to regular login page.
- The login should use the same card-form styling as the regular login, except you have some indicator it is the admin login.


# FEATURES
## New Meeting Wizard
- When a new Meeting is being created there should be some creation wizard, where the person goes through these steps
  1. Enter basic data like name and description.
  2. Enter the Agenda using the editor used in the agenda import, except there is no diff view because we know nothing came before it.
  3. Participant List: Similarly to the Agenda import, there should be a text-based import for the participant list, where each non-empty line is a participant with certain text markers being interpreted for FLINTA and Chair flags. The detected flags should be shown in real time next to the text with the same approach used in the agenda text editor.
  4. Overview Screen where you can see all the stuff entered before and confirm or deny.
## QR Codes as Dialogs
- The QR Codes for the guest login and the recovery codes should be dialog modals in the page itself instead of bringing you to another page.
- Both modals should have buttons for copying the URL the QR code shows to the clipboard.
## Receipts show actual voting behaviour
- The committments retrieved from the receipts in /receipts should contain voting behaviour. This should be shown to the user if they verify a receipt.
## Receipts in the meeting view page
- There should be a button in the votes card of the live view page, which opens a modal with all the receipts associated with this meeting to the user allowing them to quickly validate them.
## Record when agenda points are entered and left
- When activating/changing between agenda points record the time all that happens at, to be able to see when each agenda point started and how long it took.
## Admin pages are not well laid out
- The admin pages are looking quite ugly and chaotic, make them more cleaned up.