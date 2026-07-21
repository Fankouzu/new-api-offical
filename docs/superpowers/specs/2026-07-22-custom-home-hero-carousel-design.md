# Custom Home Hero Carousel Design

## Goal

Add a custom-skin-only image carousel behind the existing home hero content.

## Behavior

- Use the four supplied `image.lizh.ai` PNG URLs in their given order.
- Display one image at a time with `cover` sizing.
- Keep each image active for five seconds.
- Crossfade to the next image over one second.
- Loop from image 4 back to image 1.
- Pause rotation while the document is hidden.
- Disable the animated crossfade when reduced motion is requested.

## Visual Layering

The carousel fills the existing hero section and does not set its height. A
dark overlay sits above the images, while all existing hero text, actions, and
terminal content remain above the overlay. The temporary red review border is
removed when the carousel is installed.

## Architecture

The shared `Home` and `Hero` components gain optional background composition
points. The custom skin overrides only the home page and supplies its carousel;
the default manifest continues rendering the existing home without it.

## Verification

- Custom home renders all four image sources and advances on a five-second timer.
- Visibility changes pause and resume the timer.
- Default home renders no carousel.
- Existing hero content remains interactive and above the background.
