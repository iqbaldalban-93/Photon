package gphotos

import (
	"context"
	"fmt"

	"app/internal/protox"
)

func (a *Api) GetLibraryState(ctx context.Context, syncToken string) ([]byte, error) {
	return a.DoProto(ctx, librarySyncEndpoint, libraryStateTemplate(syncToken), true)
}

func (a *Api) GetLibraryPageInit(ctx context.Context, resumeToken string) ([]byte, error) {
	return a.DoProto(ctx, librarySyncEndpoint, libraryPageInitTemplate(resumeToken), true)
}

func (a *Api) GetLibraryPage(ctx context.Context, resumeToken, syncToken string) ([]byte, error) {
	if resumeToken == "" || syncToken == "" {
		return nil, fmt.Errorf("delta page requires resume and sync tokens")
	}
	return a.DoProto(ctx, librarySyncEndpoint, libraryPageTemplate(resumeToken, syncToken), true)
}

func libraryStateTemplate(syncToken string) []byte {
	body := map[string]any{
		"1": map[string]any{
			"1": map[string]any{
				"1": map[string]any{
					"1": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{},
					"5": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
						"3": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"7": map[string]any{},
					},
					"6": map[string]any{},
					"7": map[string]any{
						"2": map[string]any{},
					},
					"15": map[string]any{},
					"16": map[string]any{},
					"17": map[string]any{},
					"19": map[string]any{},
					"20": map[string]any{},
					"21": map[string]any{
						"5": map[string]any{
							"3": map[string]any{},
						},
						"6": map[string]any{},
					},
					"25": map[string]any{},
					"30": map[string]any{
						"2": map[string]any{},
					},
					"31": map[string]any{},
					"32": map[string]any{},
					"33": map[string]any{
						"1": map[string]any{},
					},
					"34": map[string]any{},
					"36": map[string]any{},
					"37": map[string]any{},
					"38": map[string]any{},
					"39": map[string]any{},
					"40": map[string]any{},
					"41": map[string]any{},
				},
				"5": map[string]any{
					"2": map[string]any{
						"2": map[string]any{
							"3": map[string]any{
								"2": map[string]any{},
							},
							"4": map[string]any{
								"2": map[string]any{},
								"4": map[string]any{},
							},
						},
						"4": map[string]any{
							"2": map[string]any{
								"2": 1,
							},
						},
						"5": map[string]any{
							"2": map[string]any{},
						},
						"6": 1,
					},
					"3": map[string]any{
						"2": map[string]any{
							"3": map[string]any{},
							"4": map[string]any{},
						},
						"3": map[string]any{
							"2": map[string]any{},
							"3": map[string]any{
								"2": 1,
								"3": map[string]any{},
							},
						},
						"4": map[string]any{},
						"5": map[string]any{
							"2": map[string]any{
								"2": 1,
							},
						},
						"7": map[string]any{},
					},
					"4": map[string]any{
						"2": map[string]any{
							"2": map[string]any{},
						},
					},
					"5": map[string]any{
						"1": map[string]any{
							"2": map[string]any{
								"3": map[string]any{},
								"4": map[string]any{},
							},
							"3": map[string]any{
								"2": map[string]any{},
								"3": map[string]any{
									"2": 1,
									"3": map[string]any{},
								},
							},
						},
						"3": 1,
					},
				},
				"8": map[string]any{},
				"9": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
					"4": map[string]any{
						"1": map[string]any{
							"3": map[string]any{
								"1": map[string]any{
									"1": map[string]any{
										"5": map[string]any{
											"1": map[string]any{},
										},
										"6": map[string]any{},
										"7": map[string]any{},
									},
									"2": map[string]any{},
									"3": map[string]any{
										"1": map[string]any{
											"5": map[string]any{
												"1": map[string]any{},
											},
											"6": map[string]any{},
											"7": map[string]any{},
										},
										"2": map[string]any{},
									},
								},
							},
							"4": map[string]any{
								"1": map[string]any{
									"2": map[string]any{},
								},
							},
						},
					},
				},
				"11": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"12": map[string]any{},
				"14": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"15": map[string]any{
					"1": map[string]any{},
					"4": map[string]any{},
				},
				"17": map[string]any{
					"1": map[string]any{},
					"4": map[string]any{},
				},
				"19": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"21": map[string]any{
					"1": map[string]any{},
				},
				"22": map[string]any{},
				"23": map[string]any{},
				"24": map[string]any{},
			},
			"2": map[string]any{
				"1": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{},
					"5": map[string]any{},
					"6": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
						"3": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"7": map[string]any{},
					},
					"7":  map[string]any{},
					"8":  map[string]any{},
					"10": map[string]any{},
					"12": map[string]any{},
					"13": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"15": map[string]any{
						"1": map[string]any{},
					},
					"18": map[string]any{},
				},
				"4": map[string]any{
					"1": map[string]any{},
				},
				"9": map[string]any{},
				"11": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"6": map[string]any{},
						"9": map[string]any{},
					},
				},
				"14": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"1": map[string]any{},
							"2": map[string]any{
								"2": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
							},
							"3": map[string]any{
								"4": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
								"5": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
							},
						},
						"2": map[string]any{},
					},
				},
				"17": map[string]any{},
				"18": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
					},
				},
				"20": map[string]any{
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
				},
				"22": map[string]any{},
				"23": map[string]any{},
				"24": map[string]any{},
			},
			"3": map[string]any{
				"2": map[string]any{},
				"3": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"7": map[string]any{},
					"8": map[string]any{},
					"14": map[string]any{
						"1": map[string]any{},
					},
					"16": map[string]any{},
					"17": map[string]any{
						"2": map[string]any{},
					},
					"18": map[string]any{},
					"19": map[string]any{},
					"20": map[string]any{},
					"21": map[string]any{},
					"22": map[string]any{},
					"23": map[string]any{},
					"27": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
					},
					"29": map[string]any{},
					"30": map[string]any{},
					"31": map[string]any{},
					"32": map[string]any{},
					"34": map[string]any{},
					"37": map[string]any{},
					"38": map[string]any{},
					"39": map[string]any{},
					"41": map[string]any{},
					"43": map[string]any{
						"1": map[string]any{},
					},
					"45": map[string]any{
						"1": map[string]any{
							"1": map[string]any{},
						},
					},
					"46": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"47": map[string]any{},
				},
				"4": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{
						"1": map[string]any{},
					},
					"4": map[string]any{},
					"5": map[string]any{
						"1": map[string]any{},
					},
				},
				"7":  map[string]any{},
				"12": map[string]any{},
				"13": map[string]any{},
				"14": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
						"3": map[string]any{},
						"4": map[string]any{
							"1": map[string]any{},
						},
					},
					"3": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
						"3": map[string]any{},
						"4": map[string]any{},
					},
				},
				"15": map[string]any{},
				"16": map[string]any{
					"1": map[string]any{},
				},
				"18": map[string]any{},
				"19": map[string]any{
					"4": map[string]any{
						"2": map[string]any{},
					},
					"6": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"7": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"8": map[string]any{},
					"9": map[string]any{},
				},
				"20": map[string]any{},
				"22": map[string]any{},
				"24": map[string]any{},
				"25": map[string]any{},
				"26": map[string]any{},
			},
			"6": syncToken,
			"7": 2,
			"9": map[string]any{
				"1": map[string]any{
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
				},
				"2": map[string]any{
					"3": map[string]any{
						"2": 1,
					},
				},
				"3": map[string]any{
					"2": map[string]any{},
				},
				"4": map[string]any{},
				"7": map[string]any{
					"1": map[string]any{},
				},
				"8": map[string]any{
					"1": 2,
					"2": "\x01\x02\x03\x05\x06\x07",
				},
				"9": map[string]any{},
				"11": map[string]any{
					"1": map[string]any{},
				},
			},
			"11": []any{
				1,
				2,
				6,
			},
			"12": map[string]any{
				"2": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"3": map[string]any{
					"1": map[string]any{},
				},
				"4": map[string]any{},
			},
			"13": map[string]any{},
			"15": map[string]any{
				"3": map[string]any{
					"1": 1,
				},
			},
			"18": map[string]any{
				"169945741": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"4": []any{
								2,
								1,
								6,
								8,
								10,
								15,
								18,
								13,
								17,
								19,
								14,
								20,
							},
							"5":  6,
							"6":  2,
							"7":  1,
							"8":  2,
							"11": 3,
							"12": 1,
							"13": 3,
							"15": 1,
							"16": 1,
							"17": 1,
							"18": 2,
						},
					},
				},
			},
			"19": map[string]any{
				"1": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"2": map[string]any{
					"1": []any{
						1,
						2,
						4,
						6,
						5,
						7,
					},
				},
				"3": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"5": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"6": map[string]any{
					"1": map[string]any{},
				},
				"7": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"8": map[string]any{
					"1": map[string]any{},
				},
			},
			"20": map[string]any{
				"1": 1,
				"2": "",
				"3": map[string]any{
					"1": "type.googleapis.com/photos.printing.client.PrintingPromotionSyncOptions",
					"2": map[string]any{
						"1": map[string]any{
							"4": []any{
								2,
								1,
								6,
								8,
								10,
								15,
								18,
								13,
								17,
								19,
								14,
								20,
							},
							"5":  6,
							"6":  2,
							"7":  1,
							"8":  2,
							"11": 3,
							"12": 1,
							"13": 3,
							"15": 1,
							"16": 1,
							"17": 1,
							"18": 2,
						},
					},
				},
			},
			"21": map[string]any{
				"2": map[string]any{
					"2": map[string]any{
						"4": map[string]any{},
					},
					"4": map[string]any{},
					"5": map[string]any{},
				},
				"3": map[string]any{
					"2": map[string]any{
						"1": 1,
					},
					"4": map[string]any{
						"2": map[string]any{},
					},
				},
				"5": map[string]any{
					"1": map[string]any{},
				},
				"6": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
					},
				},
				"7": map[string]any{
					"1": 2,
					"2": "\x01\x07\x08\t\n\r\x0e\x0f\x11\x13\x14\x16\x17-./01:\x06\x18267;>?@A89<GBED",
					"3": "\x01",
				},
				"8": map[string]any{
					"3": map[string]any{
						"1": map[string]any{
							"1": map[string]any{
								"2": map[string]any{
									"1": 1,
								},
								"4": map[string]any{
									"2": map[string]any{},
								},
							},
						},
						"3": map[string]any{},
					},
					"4": map[string]any{
						"1": map[string]any{},
					},
					"5": map[string]any{
						"1": map[string]any{
							"2": map[string]any{
								"1": 1,
							},
							"4": map[string]any{
								"2": map[string]any{},
							},
						},
					},
				},
				"9": map[string]any{
					"1": map[string]any{},
				},
				"10": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
					},
					"3": map[string]any{},
					"5": map[string]any{},
					"6": map[string]any{
						"1": map[string]any{},
					},
					"7":  map[string]any{},
					"9":  map[string]any{},
					"10": map[string]any{},
				},
				"11": map[string]any{},
				"12": map[string]any{},
				"13": map[string]any{},
				"14": map[string]any{},
				"16": map[string]any{
					"1": map[string]any{},
				},
			},
			"22": map[string]any{
				"1": 1,
				"2": "107818234414673686888",
			},
			"25": map[string]any{
				"1": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"1": map[string]any{},
						},
					},
				},
				"2": map[string]any{},
			},
			"26": map[string]any{},
		},
		"2": map[string]any{
			"1": map[string]any{
				"1": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
					},
					"2": map[string]any{},
				},
			},
			"2": map[string]any{},
		},
	}
	return protox.FromMap(body)
}

func libraryPageInitTemplate(resumeToken string) []byte {
	body := map[string]any{
		"1": map[string]any{
			"1": map[string]any{
				"1": map[string]any{
					"1": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{},
					"5": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
						"3": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"7": map[string]any{},
					},
					"6": map[string]any{},
					"7": map[string]any{
						"2": map[string]any{},
					},
					"15": map[string]any{},
					"16": map[string]any{},
					"17": map[string]any{},
					"19": map[string]any{},
					"20": map[string]any{},
					"21": map[string]any{
						"5": map[string]any{
							"3": map[string]any{},
						},
						"6": map[string]any{},
					},
					"25": map[string]any{},
					"30": map[string]any{
						"2": map[string]any{},
					},
					"31": map[string]any{},
					"32": map[string]any{},
					"33": map[string]any{
						"1": map[string]any{},
					},
					"34": map[string]any{},
					"36": map[string]any{},
					"37": map[string]any{},
					"38": map[string]any{},
					"39": map[string]any{},
					"40": map[string]any{},
					"41": map[string]any{},
				},
				"5": map[string]any{
					"2": map[string]any{
						"2": map[string]any{
							"3": map[string]any{
								"2": map[string]any{},
							},
							"4": map[string]any{
								"2": map[string]any{},
							},
						},
						"4": map[string]any{
							"2": map[string]any{
								"2": 1,
							},
						},
						"5": map[string]any{
							"2": map[string]any{},
						},
						"6": 1,
					},
					"3": map[string]any{
						"2": map[string]any{
							"3": map[string]any{},
							"4": map[string]any{},
						},
						"3": map[string]any{
							"2": map[string]any{},
							"3": map[string]any{
								"2": 1,
							},
						},
						"4": map[string]any{},
						"5": map[string]any{
							"2": map[string]any{
								"2": 1,
							},
						},
						"7": map[string]any{},
					},
					"4": map[string]any{
						"2": map[string]any{
							"2": map[string]any{},
						},
					},
					"5": map[string]any{
						"1": map[string]any{
							"2": map[string]any{
								"3": map[string]any{},
								"4": map[string]any{},
							},
							"3": map[string]any{
								"2": map[string]any{},
								"3": map[string]any{
									"2": 1,
								},
							},
						},
						"3": 1,
					},
				},
				"8": map[string]any{},
				"9": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
					"4": map[string]any{
						"1": map[string]any{
							"3": map[string]any{
								"1": map[string]any{
									"1": map[string]any{
										"5": map[string]any{
											"1": map[string]any{},
										},
										"6": map[string]any{},
									},
									"2": map[string]any{},
									"3": map[string]any{
										"1": map[string]any{
											"5": map[string]any{
												"1": map[string]any{},
											},
											"6": map[string]any{},
										},
										"2": map[string]any{},
									},
								},
							},
							"4": map[string]any{
								"1": map[string]any{
									"2": map[string]any{},
								},
							},
						},
					},
				},
				"11": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"12": map[string]any{},
				"14": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"15": map[string]any{
					"1": map[string]any{},
					"4": map[string]any{},
				},
				"17": map[string]any{
					"1": map[string]any{},
					"4": map[string]any{},
				},
				"19": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"22": map[string]any{},
				"23": map[string]any{},
			},
			"2": map[string]any{
				"1": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{},
					"5": map[string]any{},
					"6": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
						"3": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"7": map[string]any{},
					},
					"7":  map[string]any{},
					"8":  map[string]any{},
					"10": map[string]any{},
					"12": map[string]any{},
					"13": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"15": map[string]any{
						"1": map[string]any{},
					},
					"18": map[string]any{},
				},
				"4": map[string]any{
					"1": map[string]any{},
				},
				"9": map[string]any{},
				"11": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"6": map[string]any{},
						"9": map[string]any{},
					},
				},
				"14": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"1": map[string]any{},
							"2": map[string]any{
								"2": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
							},
							"3": map[string]any{
								"4": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
								"5": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
							},
						},
						"2": map[string]any{},
					},
				},
				"17": map[string]any{},
				"18": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
					},
				},
				"20": map[string]any{
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
				},
				"23": map[string]any{},
			},
			"3": map[string]any{
				"2": map[string]any{},
				"3": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"7": map[string]any{},
					"8": map[string]any{},
					"14": map[string]any{
						"1": map[string]any{},
					},
					"16": map[string]any{},
					"17": map[string]any{
						"2": map[string]any{},
					},
					"18": map[string]any{},
					"19": map[string]any{},
					"20": map[string]any{},
					"21": map[string]any{},
					"22": map[string]any{},
					"23": map[string]any{},
					"27": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
					},
					"29": map[string]any{},
					"30": map[string]any{},
					"31": map[string]any{},
					"32": map[string]any{},
					"34": map[string]any{},
					"37": map[string]any{},
					"38": map[string]any{},
					"39": map[string]any{},
					"41": map[string]any{},
				},
				"4": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{},
				},
				"7":  map[string]any{},
				"12": map[string]any{},
				"13": map[string]any{},
				"14": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
						"3": map[string]any{},
						"4": map[string]any{
							"1": map[string]any{},
						},
					},
					"3": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
						"3": map[string]any{},
						"4": map[string]any{},
					},
				},
				"15": map[string]any{},
				"16": map[string]any{
					"1": map[string]any{},
				},
				"18": map[string]any{},
				"19": map[string]any{
					"4": map[string]any{
						"2": map[string]any{},
					},
					"6": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"7": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"8": map[string]any{},
				},
				"20": map[string]any{},
				"24": map[string]any{},
				"25": map[string]any{},
			},
			"4": resumeToken,
			"7": 2,
			"9": map[string]any{
				"1": map[string]any{
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
				},
				"2": map[string]any{
					"3": map[string]any{
						"2": 1,
					},
				},
				"3": map[string]any{
					"2": map[string]any{},
				},
				"4": map[string]any{},
				"7": map[string]any{
					"1": map[string]any{},
				},
				"8": map[string]any{
					"1": 2,
					"2": "\x01\x02\x03\x05\x06",
				},
				"9": map[string]any{},
			},
			"11": []any{
				1,
				2,
			},
			"12": map[string]any{
				"2": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"3": map[string]any{
					"1": map[string]any{},
				},
				"4": map[string]any{},
			},
			"13": map[string]any{},
			"15": map[string]any{
				"3": map[string]any{
					"1": 1,
				},
			},
			"18": map[string]any{
				"169945741": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"4": []any{
								2,
								1,
								6,
								8,
								10,
								15,
								18,
								13,
								17,
								19,
								14,
								20,
							},
							"5":  6,
							"6":  2,
							"7":  1,
							"8":  2,
							"11": 3,
							"12": 1,
							"13": 3,
							"15": 1,
							"16": 1,
							"17": 1,
							"18": 2,
						},
					},
				},
			},
			"19": map[string]any{
				"1": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"2": map[string]any{
					"1": []any{
						1,
						2,
						4,
						6,
						5,
						7,
					},
				},
				"3": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"5": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"6": map[string]any{
					"1": map[string]any{},
				},
				"7": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"8": map[string]any{
					"1": map[string]any{},
				},
			},
			"20": map[string]any{
				"1": 1,
				"3": map[string]any{
					"1": "type.googleapis.com/photos.printing.client.PrintingPromotionSyncOptions",
					"2": map[string]any{
						"1": map[string]any{
							"4": []any{
								2,
								1,
								6,
								8,
								10,
								15,
								18,
								13,
								17,
								19,
								14,
								20,
							},
							"5":  6,
							"6":  2,
							"7":  1,
							"8":  2,
							"11": 3,
							"12": 1,
							"13": 3,
							"15": 1,
							"16": 1,
							"17": 1,
							"18": 2,
						},
					},
				},
			},
			"21": map[string]any{
				"2": map[string]any{
					"2": map[string]any{},
					"4": map[string]any{},
					"5": map[string]any{},
				},
				"3": map[string]any{
					"2": map[string]any{
						"1": 1,
					},
				},
				"5": map[string]any{
					"1": map[string]any{},
				},
				"6": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
					},
				},
				"7": map[string]any{
					"1": 2,
					"2": "\x01\x07\x08\t\n\r\x0e\x0f\x11\x13\x14\x16\x17-./01:\x06\x18267;>?@A89<",
					"3": "\x01",
				},
				"8": map[string]any{
					"3": map[string]any{
						"1": map[string]any{
							"1": map[string]any{
								"2": map[string]any{
									"1": 1,
								},
							},
						},
					},
					"4": map[string]any{
						"1": map[string]any{},
					},
				},
				"9": map[string]any{
					"1": map[string]any{},
				},
				"10": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
					},
					"3": map[string]any{},
					"5": map[string]any{},
					"6": map[string]any{
						"1": map[string]any{},
					},
					"7":  map[string]any{},
					"9":  map[string]any{},
					"10": map[string]any{},
				},
				"11": map[string]any{},
				"12": map[string]any{},
				"13": map[string]any{},
			},
			"22": map[string]any{
				"1": 2,
			},
			"25": map[string]any{
				"1": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"1": map[string]any{},
						},
					},
				},
				"2": map[string]any{},
			},
		},
		"2": map[string]any{
			"1": map[string]any{
				"1": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
					},
					"2": map[string]any{},
				},
			},
			"2": map[string]any{},
		},
	}
	return protox.FromMap(body)
}

func libraryPageTemplate(resumeToken, syncToken string) []byte {
	body := map[string]any{
		"1": map[string]any{
			"1": map[string]any{
				"1": map[string]any{
					"1": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{},
					"5": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
						"3": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"7": map[string]any{},
					},
					"6": map[string]any{},
					"7": map[string]any{
						"2": map[string]any{},
					},
					"15": map[string]any{},
					"16": map[string]any{},
					"17": map[string]any{},
					"19": map[string]any{},
					"20": map[string]any{},
					"21": map[string]any{
						"5": map[string]any{
							"3": map[string]any{},
						},
						"6": map[string]any{},
					},
					"25": map[string]any{},
					"30": map[string]any{
						"2": map[string]any{},
					},
					"31": map[string]any{},
					"32": map[string]any{},
					"33": map[string]any{
						"1": map[string]any{},
					},
					"34": map[string]any{},
					"36": map[string]any{},
					"37": map[string]any{},
					"38": map[string]any{},
					"39": map[string]any{},
					"40": map[string]any{},
					"41": map[string]any{},
				},
				"5": map[string]any{
					"2": map[string]any{
						"2": map[string]any{
							"3": map[string]any{
								"2": map[string]any{},
							},
							"4": map[string]any{
								"2": map[string]any{},
							},
						},
						"4": map[string]any{
							"2": map[string]any{
								"2": 1,
							},
						},
						"5": map[string]any{
							"2": map[string]any{},
						},
						"6": 1,
					},
					"3": map[string]any{
						"2": map[string]any{
							"3": map[string]any{},
							"4": map[string]any{},
						},
						"3": map[string]any{
							"2": map[string]any{},
							"3": map[string]any{
								"2": 1,
							},
						},
						"4": map[string]any{},
						"5": map[string]any{
							"2": map[string]any{
								"2": 1,
							},
						},
						"7": map[string]any{},
					},
					"4": map[string]any{
						"2": map[string]any{
							"2": map[string]any{},
						},
					},
					"5": map[string]any{
						"1": map[string]any{
							"2": map[string]any{
								"3": map[string]any{},
								"4": map[string]any{},
							},
							"3": map[string]any{
								"2": map[string]any{},
								"3": map[string]any{
									"2": 1,
								},
							},
						},
						"3": 1,
					},
				},
				"8": map[string]any{},
				"9": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
					"4": map[string]any{
						"1": map[string]any{
							"3": map[string]any{
								"1": map[string]any{
									"1": map[string]any{
										"5": map[string]any{
											"1": map[string]any{},
										},
										"6": map[string]any{},
									},
									"2": map[string]any{},
									"3": map[string]any{
										"1": map[string]any{
											"5": map[string]any{
												"1": map[string]any{},
											},
											"6": map[string]any{},
										},
										"2": map[string]any{},
									},
								},
							},
							"4": map[string]any{
								"1": map[string]any{
									"2": map[string]any{},
								},
							},
						},
					},
				},
				"11": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"12": map[string]any{},
				"14": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"15": map[string]any{
					"1": map[string]any{},
					"4": map[string]any{},
				},
				"17": map[string]any{
					"1": map[string]any{},
					"4": map[string]any{},
				},
				"19": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{
						"2": map[string]any{
							"1": 1,
							"2": 2,
						},
					},
				},
				"22": map[string]any{},
				"23": map[string]any{},
			},
			"2": map[string]any{
				"1": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{},
					"5": map[string]any{},
					"6": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
						"3": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"7": map[string]any{},
					},
					"7":  map[string]any{},
					"8":  map[string]any{},
					"10": map[string]any{},
					"12": map[string]any{},
					"13": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"15": map[string]any{
						"1": map[string]any{},
					},
					"18": map[string]any{},
				},
				"4": map[string]any{
					"1": map[string]any{},
				},
				"9": map[string]any{},
				"11": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
						"4": map[string]any{},
						"5": map[string]any{},
						"6": map[string]any{},
						"9": map[string]any{},
					},
				},
				"14": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"1": map[string]any{},
							"2": map[string]any{
								"2": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
							},
							"3": map[string]any{
								"4": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
								"5": map[string]any{
									"1": map[string]any{
										"1": map[string]any{},
									},
									"3": map[string]any{},
								},
							},
						},
						"2": map[string]any{},
					},
				},
				"17": map[string]any{},
				"18": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
					},
				},
				"20": map[string]any{
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
				},
				"23": map[string]any{},
			},
			"3": map[string]any{
				"2": map[string]any{},
				"3": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"7": map[string]any{},
					"8": map[string]any{},
					"14": map[string]any{
						"1": map[string]any{},
					},
					"16": map[string]any{},
					"17": map[string]any{
						"2": map[string]any{},
					},
					"18": map[string]any{},
					"19": map[string]any{},
					"20": map[string]any{},
					"21": map[string]any{},
					"22": map[string]any{},
					"23": map[string]any{},
					"27": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
					},
					"29": map[string]any{},
					"30": map[string]any{},
					"31": map[string]any{},
					"32": map[string]any{},
					"34": map[string]any{},
					"37": map[string]any{},
					"38": map[string]any{},
					"39": map[string]any{},
					"41": map[string]any{},
				},
				"4": map[string]any{
					"2": map[string]any{},
					"3": map[string]any{},
					"4": map[string]any{},
				},
				"7":  map[string]any{},
				"12": map[string]any{},
				"13": map[string]any{},
				"14": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
						"3": map[string]any{},
						"4": map[string]any{
							"1": map[string]any{},
						},
					},
					"3": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{
							"1": map[string]any{},
						},
						"3": map[string]any{},
						"4": map[string]any{},
					},
				},
				"15": map[string]any{},
				"16": map[string]any{
					"1": map[string]any{},
				},
				"18": map[string]any{},
				"19": map[string]any{
					"4": map[string]any{
						"2": map[string]any{},
					},
					"6": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"7": map[string]any{
						"2": map[string]any{},
						"3": map[string]any{},
					},
					"8": map[string]any{},
				},
				"20": map[string]any{},
				"24": map[string]any{},
				"25": map[string]any{},
			},
			"4": resumeToken,
			"6": syncToken,
			"7": 2,
			"9": map[string]any{
				"1": map[string]any{
					"2": map[string]any{
						"1": map[string]any{},
						"2": map[string]any{},
					},
				},
				"2": map[string]any{
					"3": map[string]any{
						"2": 1,
					},
				},
				"3": map[string]any{
					"2": map[string]any{},
				},
				"4": map[string]any{},
				"7": map[string]any{
					"1": map[string]any{},
				},
				"8": map[string]any{
					"1": 2,
					"2": "\x01\x02\x03\x05\x06",
				},
				"9": map[string]any{},
			},
			"11": []any{
				1,
				2,
			},
			"12": map[string]any{
				"2": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"3": map[string]any{
					"1": map[string]any{},
				},
				"4": map[string]any{},
			},
			"13": map[string]any{},
			"15": map[string]any{
				"3": map[string]any{
					"1": 1,
				},
			},
			"18": map[string]any{
				"169945741": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"4": []any{
								2,
								1,
								6,
								8,
								10,
								15,
								18,
								13,
								17,
								19,
								14,
								20,
							},
							"5":  6,
							"6":  2,
							"7":  1,
							"8":  2,
							"11": 3,
							"12": 1,
							"13": 3,
							"15": 1,
							"16": 1,
							"17": 1,
							"18": 2,
						},
					},
				},
			},
			"19": map[string]any{
				"1": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"2": map[string]any{
					"1": []any{
						1,
						2,
						4,
						6,
						5,
						7,
					},
				},
				"3": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"5": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"6": map[string]any{
					"1": map[string]any{},
				},
				"7": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{},
				},
				"8": map[string]any{
					"1": map[string]any{},
				},
			},
			"20": map[string]any{
				"1": 1,
				"2": "AH_uQ41bEgartCAb9ZVh48fOzHLvaA7xJy_EHlv_4kR6Q7xI4Bol3igCVJ6HJ_VViRfrDrBJB5EQ",
				"3": map[string]any{
					"1": "type.googleapis.com/photos.printing.client.PrintingPromotionSyncOptions",
					"2": map[string]any{
						"1": map[string]any{
							"4": []any{
								2,
								1,
								6,
								8,
								10,
								15,
								18,
								13,
								17,
								19,
								14,
								20,
							},
							"5":  6,
							"6":  2,
							"7":  1,
							"8":  2,
							"11": 3,
							"12": 1,
							"13": 3,
							"15": 1,
							"16": 1,
							"17": 1,
							"18": 2,
						},
					},
				},
			},
			"21": map[string]any{
				"2": map[string]any{
					"2": map[string]any{},
					"4": map[string]any{},
					"5": map[string]any{},
				},
				"3": map[string]any{
					"2": map[string]any{
						"1": 1,
					},
				},
				"5": map[string]any{
					"1": map[string]any{},
				},
				"6": map[string]any{
					"1": map[string]any{},
					"2": map[string]any{
						"1": map[string]any{},
					},
				},
				"7": map[string]any{
					"1": 2,
					"2": "\x01\x07\x08\t\n\r\x0e\x0f\x11\x13\x14\x16\x17-./01:\x06\x18267;>?@A89<",
					"3": "\x01",
				},
				"8": map[string]any{
					"3": map[string]any{
						"1": map[string]any{
							"1": map[string]any{
								"2": map[string]any{
									"1": 1,
								},
							},
						},
					},
					"4": map[string]any{
						"1": map[string]any{},
					},
				},
				"9": map[string]any{
					"1": map[string]any{},
				},
				"10": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
					},
					"3": map[string]any{},
					"5": map[string]any{},
					"6": map[string]any{
						"1": map[string]any{},
					},
					"7":  map[string]any{},
					"9":  map[string]any{},
					"10": map[string]any{},
				},
				"11": map[string]any{},
				"12": map[string]any{},
				"13": map[string]any{},
			},
			"22": map[string]any{
				"1": 2,
			},
			"25": map[string]any{
				"1": map[string]any{
					"1": map[string]any{
						"1": map[string]any{
							"1": map[string]any{},
						},
					},
				},
				"2": map[string]any{},
			},
		},
		"2": map[string]any{
			"1": map[string]any{
				"1": map[string]any{
					"1": map[string]any{
						"1": map[string]any{},
					},
					"2": map[string]any{},
				},
			},
			"2": map[string]any{},
		},
	}
	return protox.FromMap(body)
}
