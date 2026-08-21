package model

type CategoryID string

const (
	CatEnvironment CategoryID = "environment"
	CatElderly     CategoryID = "elderly"
	CatEducation   CategoryID = "education"
	CatCommunity   CategoryID = "community"
	CatEmergency   CategoryID = "emergency"
	CatCulture     CategoryID = "culture"
	CatSports      CategoryID = "sports"
	CatOther       CategoryID = "other"
)

type Category struct {
	ID    CategoryID `json:"id"`
	Name  string     `json:"name"`
	Order int        `json:"order"`
}

func AllCategories() []Category {
	return []Category{
		{ID: CatEnvironment, Name: "环保", Order: 1},
		{ID: CatElderly, Name: "助老", Order: 2},
		{ID: CatEducation, Name: "支教", Order: 3},
		{ID: CatCommunity, Name: "社区服务", Order: 4},
		{ID: CatEmergency, Name: "应急", Order: 5},
		{ID: CatCulture, Name: "文化", Order: 6},
		{ID: CatSports, Name: "体育", Order: 7},
		{ID: CatOther, Name: "其他", Order: 8},
	}
}

func ValidCategory(id CategoryID) bool {
	for _, c := range AllCategories() {
		if c.ID == id {
			return true
		}
	}
	return false
}

func CategoryName(id CategoryID) string {
	for _, c := range AllCategories() {
		if c.ID == id {
			return c.Name
		}
	}
	return string(id)
}
