package prefs

func SetUseUseHorizontalSplit(use bool) error {
	config := GetGlobalConfig()
	config.Spec.General.UseHorizontalSplit = use
	return UpdateGlobalConfig(config)
}
