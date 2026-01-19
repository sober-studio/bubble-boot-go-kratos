package main

import (
	"flag"
	"os"

	"github.com/sober-studio/bubble-boot-go-kratos/internal/conf"
	"github.com/sober-studio/bubble-boot-go-kratos/internal/data"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

var flagconf string

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()

	logger := log.NewStdLogger(os.Stdout)

	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	db := data.NewDB(bc.Data, logger)

	g := gen.NewGenerator(gen.Config{
		// 生成代码存放目录
		OutPath: "internal/data/query",
		// 模型定义存放目录
		ModelPkgPath: "internal/data/model",
		// 模式
		Mode: gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
		// 配置字段类型，对 text[] 数组使用 pq.StringArray
		FieldWithTypeTag: true,
		// 开启此项：只要数据库字段定义没有 NOT NULL，生成的 Go 字段就会是指针
		FieldNullable: true,
	})

	baseModelOpt := []gen.ModelOpt{
		gen.FieldNew("", "BaseModel", field.Tag{
			"gorm": "embedded",
		}),
		gen.FieldIgnore("id"),
		gen.FieldIgnore("created_at"),
		gen.FieldIgnore("updated_at"),
		gen.FieldIgnore("deleted_at"),
	}

	// 显式设置生成器使用的数据库连接
	// 必须在调用 GenerateModel 之前调用 UseDB，否则会报错：UseDB() is necessary to generate model struct
	g.UseDB(db)

	// 读取数据库中的所有表名
	tableList, err := db.Migrator().GetTables()
	if err != nil {
		panic(err)
	}

	h := log.NewHelper(logger)

	// 遍历所有表，为每个表生成模型时应用 baseModelOpt
	for _, tableName := range tableList {
		// 过滤掉系统表或迁移表
		if tableName == "schema_migrations" {
			continue
		}

		// 检查表结构是否符合 BaseModel 的要求（包含 id, created_at, updated_at, deleted_at）
		// 如果符合，则嵌入 BaseModel 并忽略重复字段
		// 如果不符合（例如关联表或不带软删除的表），则按默认方式生成并输出警告
		var missingColumns []string
		if !db.Migrator().HasColumn(tableName, "id") {
			missingColumns = append(missingColumns, "id")
		}
		if !db.Migrator().HasColumn(tableName, "created_at") {
			missingColumns = append(missingColumns, "created_at")
		}
		if !db.Migrator().HasColumn(tableName, "updated_at") {
			missingColumns = append(missingColumns, "updated_at")
		}
		if !db.Migrator().HasColumn(tableName, "deleted_at") {
			missingColumns = append(missingColumns, "deleted_at")
		}

		if len(missingColumns) == 0 {
			g.ApplyBasic(g.GenerateModel(tableName, baseModelOpt...))
		} else {
			h.Warnf("Table '%s' is missing columns %v. Generating without BaseModel embedding.", tableName, missingColumns)
			g.ApplyBasic(g.GenerateModel(tableName))
		}
	}

	// 不再使用 GenerateAllTable，因为它不支持自定义 ModelOpt 列表
	// g.GenerateAllTable()
	g.Execute()
}
