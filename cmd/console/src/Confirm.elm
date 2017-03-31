module Confirm exposing (dialog)

import Native.Confirm

dialog : String -> Result () Bool
dialog question =
    Native.Confirm.dialog