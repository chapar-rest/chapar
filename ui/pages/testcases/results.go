package testcases

import (
	"fmt"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/chapar-rest/chapar/internal/domain"
	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type ResultsPanel struct {
	result *domain.TestResult
	theme  *chapartheme.Theme

	stepExpandables map[int]*widget.Clickable
}

func NewResultsPanel(theme *chapartheme.Theme) *ResultsPanel {
	return &ResultsPanel{
		theme:           theme,
		stepExpandables: make(map[int]*widget.Clickable),
	}
}

func (r *ResultsPanel) UpdateResult(result *domain.TestResult) {
	r.result = result
	// Ensure we have expandables for all steps
	for i := range result.StepResults {
		if _, ok := r.stepExpandables[i]; !ok {
			r.stepExpandables[i] = &widget.Clickable{}
		}
	}
}

func (r *ResultsPanel) ClearResult() {
	r.result = nil
	r.stepExpandables = make(map[int]*widget.Clickable)
}

func (r *ResultsPanel) HasResult() bool {
	return r.result != nil
}

func (r *ResultsPanel) Layout(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	if r.result == nil {
		return layout.Dimensions{}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Header
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Bottom: unit.Dp(5),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.layoutHeader(gtx, th)
			})
		}),
		// Steps list
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{
				Top: unit.Dp(5),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return r.layoutSteps(gtx, th)
			})
		}),
	)
}

func (r *ResultsPanel) layoutHeader(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	statusText := r.result.Status
	statusColor := th.Fg

	switch r.result.Status {
	case domain.TestResultStatusPassed:
		statusColor = chapartheme.LightGreen
		statusText = "✓ Passed"
	case domain.TestResultStatusFailed:
		statusColor = th.ErrorColor
		statusText = "✗ Failed"
	case domain.TestResultStatusRunning:
		statusColor = th.WarningColor
		statusText = "⏳ Running"
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Label(th.Material(), th.TextSize+2, fmt.Sprintf("Test: %s", r.result.TestCaseName))
			label.Color = th.Fg
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.Label(th.Material(), th.TextSize, statusText)
						label.Color = statusColor
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						summary := fmt.Sprintf("(%d/%d steps passed, %s)",
							r.result.PassedSteps,
							r.result.TotalSteps,
							r.result.Duration.Round(1e6))
						label := material.Label(th.Material(), th.TextSize-1, summary)
						label.Color = th.Fg
						return label.Layout(gtx)
					}),
				)
			})
		}),
	)
}

func (r *ResultsPanel) layoutSteps(gtx layout.Context, th *chapartheme.Theme) layout.Dimensions {
	list := layout.List{Axis: layout.Vertical}
	return list.Layout(gtx, len(r.result.StepResults), func(gtx layout.Context, i int) layout.Dimensions {
		return layout.Inset{
			Top:    unit.Dp(5),
			Bottom: unit.Dp(5),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return r.layoutStep(gtx, th, i)
		})
	})
}

func (r *ResultsPanel) layoutStep(gtx layout.Context, th *chapartheme.Theme, index int) layout.Dimensions {
	step := r.result.StepResults[index]

	statusIcon := "⏳"
	statusColor := th.WarningColor
	switch step.Status {
	case domain.TestResultStatusPassed:
		statusIcon = "✓"
		statusColor = chapartheme.LightGreen
	case domain.TestResultStatusFailed:
		statusIcon = "✗"
		statusColor = th.ErrorColor
	}

	border := widget.Border{
		Color:        th.BorderColor,
		Width:        unit.Dp(1),
		CornerRadius: unit.Dp(4),
	}

	return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// Step header
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Label(th.Material(), th.TextSize+2, statusIcon)
							label.Color = statusColor
							return label.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Label(th.Material(), th.TextSize, step.StepName)
							label.Color = th.Fg
							return label.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							timing := fmt.Sprintf("(%s)", step.Duration.Round(1e6))
							label := material.Label(th.Material(), th.TextSize-1, timing)
							label.Color = th.Fg
							return label.Layout(gtx)
						}),
					)
				}),
				// Request details
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if step.RequestMethod == "" && step.RequestURL == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						requestInfo := fmt.Sprintf("%s %s → %d",
							step.RequestMethod,
							step.RequestURL,
							step.ResponseStatus)
						label := material.Label(th.Material(), th.TextSize-1, requestInfo)
						label.Color = th.Fg
						return label.Layout(gtx)
					})
				}),
				// Error message
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if step.Error == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						label := material.Label(th.Material(), th.TextSize-1, fmt.Sprintf("Error: %s", step.Error))
						label.Color = th.ErrorColor
						return label.Layout(gtx)
					})
				}),
				// Assertion results
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(step.AssertionResults) == 0 {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.layoutAssertions(gtx, th, step.AssertionResults)
					})
				}),
				// Hook errors
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if len(step.BeforeHookErrors) == 0 && len(step.AfterHookErrors) == 0 {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return r.layoutHookErrors(gtx, th, step)
					})
				}),
			)
		})
	})
}

func (r *ResultsPanel) layoutAssertions(gtx layout.Context, th *chapartheme.Theme, assertions []domain.AssertionResult) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			label := material.Label(th.Material(), th.TextSize-1, "Assertions:")
			label.Color = th.Fg
			return label.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(3), Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				list := layout.List{Axis: layout.Vertical}
				return list.Layout(gtx, len(assertions), func(gtx layout.Context, i int) layout.Dimensions {
					assertion := assertions[i]
					assertionIcon := "✓"
					assertionColor := chapartheme.LightGreen
					if !assertion.Passed {
						assertionIcon = "✗"
						assertionColor = th.ErrorColor
					}

					return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								label := material.Label(th.Material(), th.TextSize-1, assertionIcon)
								label.Color = assertionColor
								return label.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								text := fmt.Sprintf("%s: %s", assertion.Field, assertion.Message)
								if !assertion.Passed && assertion.Expected != "" {
									text = fmt.Sprintf("%s (expected: %s, got: %s)",
										assertion.Field, assertion.Expected, assertion.Actual)
								}
								label := material.Label(th.Material(), th.TextSize-2, text)
								label.Color = th.Fg
								return label.Layout(gtx)
							}),
						)
					})
				})
			})
		}),
	)
}

func (r *ResultsPanel) layoutHookErrors(gtx layout.Context, th *chapartheme.Theme, step domain.TestStepResult) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(step.BeforeHookErrors) == 0 {
				return layout.Dimensions{}
			}
			return layout.Inset{Bottom: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.Label(th.Material(), th.TextSize-1, "Before hook errors:")
						label.Color = th.ErrorColor
						return label.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(2), Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							list := layout.List{Axis: layout.Vertical}
							return list.Layout(gtx, len(step.BeforeHookErrors), func(gtx layout.Context, i int) layout.Dimensions {
								label := material.Label(th.Material(), th.TextSize-2, fmt.Sprintf("• %s", step.BeforeHookErrors[i]))
								label.Color = th.ErrorColor
								return label.Layout(gtx)
							})
						})
					}),
				)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(step.AfterHookErrors) == 0 {
				return layout.Dimensions{}
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := material.Label(th.Material(), th.TextSize-1, "After hook errors:")
					label.Color = th.ErrorColor
					return label.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(2), Left: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						list := layout.List{Axis: layout.Vertical}
						return list.Layout(gtx, len(step.AfterHookErrors), func(gtx layout.Context, i int) layout.Dimensions {
							label := material.Label(th.Material(), th.TextSize-2, fmt.Sprintf("• %s", step.AfterHookErrors[i]))
							label.Color = th.ErrorColor
							return label.Layout(gtx)
						})
					})
				}),
			)
		}),
	)
}
