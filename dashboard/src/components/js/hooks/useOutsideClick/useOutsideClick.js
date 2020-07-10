import { useEffect, useCallback } from "react";

/**
 * Detects when there's a click event outside the given element
 *
 * @param {function} callback Method to execute when users click outside the element
 * @param {Array[object]} list of ref React objects that references an element in the DOM
 * @param {boolean} enabled controls when the even listener should be added or not
 */
const useOutsideClick = (callback, refs, enabled = true) => {
  const memoizeClick = useCallback(
    e => {
      const clickedOutside =
        refs &&
        refs.every(ref => {
          return ref.current && !ref.current.contains(e.target);
        });

      if (clickedOutside) {
        callback();
      }
    },
    [callback, refs],
  );

  useEffect(() => {
    if (enabled) {
      document.addEventListener("mousedown", memoizeClick);
    }
    return () => document.removeEventListener("mousedown", memoizeClick);
  }, [memoizeClick, enabled]);
};

export default useOutsideClick;
