package uiv2

import (
	"fmt"
	"log"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	coreevents "cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	appevents "github.com/chapar-rest/chapar/internal/events"
	"github.com/chapar-rest/chapar/internal/repository"
)

func NewAppBar(b *core.Body, repo repository.RepositoryV2) error {
	workspaceItems, err := prepareWorkspaceItems(repo)
	if err != nil {
		return err
	}

	environmentItems, err := prepareEnvironmentItems(repo)
	if err != nil {
		return err
	}

	var workspaceChooser *core.Chooser
	var environmentChooser *core.Chooser

	b.AddTopBar(func(bar *core.Frame) {
		bar.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Padding.Set(units.Dp(4), units.Dp(8))
			s.Gap.Set(units.Dp(2))
			s.Justify.Content = styles.SpaceAround
			s.Align.Items = styles.Center
			s.Border.Style.Bottom = styles.BorderSolid
			s.Border.Width.Bottom = units.Dp(1)
			s.Border.Color.Bottom = colors.Scheme.OutlineVariant
		})

		left := core.NewFrame(bar)
		left.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Justify.Content = styles.Start
		})

		core.NewText(left).
			SetText("Chapar").
			SetType(core.TextTitleMedium).
			Styler(func(s *styles.Style) {
				s.Padding.Set(units.Dp(4), units.Dp(12))
				s.Min.X.Dp(90)
				s.Max.X.Dp(90)
			})

		workspaceChooser = core.NewChooser(left).SetItems(workspaceItems...)
		workspaceChooser.Styler(func(s *styles.Style) {
			s.Min.X.Dp(160)
			s.Max.X.Dp(160)
		})
		workspaceChooser.OnChange(func(e coreevents.Event) {
			selectedName := fmt.Sprint(workspaceChooser.CurrentItem.Value)
			workspaces, err := repo.LoadWorkspaces()
			if err != nil {
				log.Println(err)
				return
			}

			for _, workspace := range workspaces {
				if workspace.GetName() == selectedName {
					appevents.WorkspaceSelectedTopic.Publish(workspace)
					return
				}
			}
		})

		right := core.NewFrame(bar)
		right.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Justify.Content = styles.End
		})

		environmentChooser = core.NewChooser(right).SetItems(environmentItems...)
		environmentChooser.Styler(func(s *styles.Style) {
			s.Min.X.Dp(160)
			s.Max.X.Dp(160)
		})
		environmentChooser.OnChange(func(e coreevents.Event) {
			selectedName := fmt.Sprint(environmentChooser.CurrentItem.Value)
			environments, err := repo.LoadEnvironments()
			if err != nil {
				log.Println(err)
				return
			}

			for _, environment := range environments {
				if environment.GetName() == selectedName {
					appevents.EnvironmentSelectedTopic.Publish(environment)
					return
				}
			}
		})
	})

	wkSub := appevents.WorkspaceChangeTopic.Subscribe()
	envSub := appevents.EnvironmentChangeTopic.Subscribe()

	b.OnClose(func(e coreevents.Event) {
		wkSub.Unsubscribe()
		envSub.Unsubscribe()
	})

	go func() {
		for range wkSub.C {
			updatedItems, err := prepareWorkspaceItems(repo)
			if err != nil {
				log.Println(err)
				continue
			}

			system.TheApp.RunOnMain(func() {
				workspaceChooser.SetItems(updatedItems...)
				workspaceChooser.Update()
			})
		}
	}()

	go func() {
		for range envSub.C {
			updatedItems, err := prepareEnvironmentItems(repo)
			if err != nil {
				log.Println(err)
				continue
			}

			system.TheApp.RunOnMain(func() {
				environmentChooser.SetItems(updatedItems...)
				environmentChooser.Update()
			})
		}
	}()

	return nil
}

func prepareWorkspaceItems(repo repository.RepositoryV2) ([]core.ChooserItem, error) {
	workspaces, err := repo.LoadWorkspaces()
	if err != nil {
		return nil, err
	}

	workspaceItems := make([]core.ChooserItem, 0, len(workspaces))
	for _, workspace := range workspaces {
		workspaceItems = append(workspaceItems, core.ChooserItem{Value: workspace.GetName(), Text: workspace.GetName()})
	}
	return workspaceItems, nil
}

func prepareEnvironmentItems(repo repository.RepositoryV2) ([]core.ChooserItem, error) {
	environments, err := repo.LoadEnvironments()
	if err != nil {
		return nil, err
	}

	environmentItems := make([]core.ChooserItem, 0, len(environments))
	for _, environment := range environments {
		environmentItems = append(environmentItems, core.ChooserItem{Value: environment.GetName(), Text: environment.GetName()})
	}
	return environmentItems, nil
}
