TODOs / FIXMEs not quite super important and maybe not yet issues:

* Allow absence of sticky bit via option, if not supported by FS
* Check for permissions and set the correctly
* Implement deletion across partitions if required somehow.
* Decide whether we early exit on errors or try to delete all paths. Later, this can be a setting. The decision should be documented.
* Figure out, whether this should only empty whatever the spec would also
  demanding deleting into or all reachable trashbins. An alternative would
  be to clear the topdir trash, if available and the user trash. Considering
  the nature of wastebasket, it would probably be best to clear the user
  trash. In the future, we could optionally allow passing a path here, so the
  user can define a custom path or clearing options.
  
  This could have a format where you can define different options for
  different platforms, such as:
  
  ```go
  wastebasket.Empty(
    wastebasket.Pattern("*.txt"),
    nix.ClearUserTrashbin(),
    darwin.ClearAllAvailableTrashbins(),
  )
  ```