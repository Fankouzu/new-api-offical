# Custom Home Hero Boundary Design

## Goal

Show a temporary visual boundary for the future custom-skin home hero so its
vertical extent can be reviewed before visual design begins.

## Boundary

- Applies only to the `custom` skin home page.
- Starts at the top of the document and includes the site navigation.
- Has an exact height of `850px`.
- Uses a conspicuous red dashed border.
- Does not capture pointer events or alter existing content layout.
- Scrolls with the document rather than remaining fixed to the viewport.

## Implementation Boundary

The marker is a temporary presentation layer owned by the custom skin. It must
not modify the default skin, shared navigation layout, home business logic, or
the eventual hero component structure.

## Verification

- The custom home page displays the boundary from document y=0 through y=850.
- Other custom public routes do not display it.
- The default skin never displays it.
- Links and controls inside the marked area remain interactive.
