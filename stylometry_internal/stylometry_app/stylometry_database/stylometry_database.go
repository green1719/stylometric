package database

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"stylometry/stylometry_internal/stylometry_app/stylometry_analysis"
	"stylometry/stylometry_internal/stylometry_app/stylometry_model"
)

// NewDatabase создаёт подключение к БД, миграции и стартовые данные.
func NewDatabase() (*gorm.DB, error) {
	var stylometry_db *gorm.DB
	var stylometry_err error
	for stylometry_attempt := 1; stylometry_attempt <= 10; stylometry_attempt++ {
		stylometry_db, stylometry_err = gorm.Open(postgres.Open(stylometryBuildDSN()), &gorm.Config{})
		if stylometry_err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if stylometry_err != nil {
		return nil, stylometry_err
	}

	if stylometry_err = stylometryMigrateLegacySchema(stylometry_db); stylometry_err != nil {
		return nil, stylometry_err
	}
	if stylometry_err = stylometryMigrateSchemaNames(stylometry_db); stylometry_err != nil {
		return nil, stylometry_err
	}

	stylometry_err = stylometry_db.AutoMigrate(
		&model.User{},
		&model.Sample{},
		&model.Like{},
	)
	if stylometry_err != nil {
		return nil, stylometry_err
	}
	if stylometry_err = stylometryEnsureDraftIndex(stylometry_db); stylometry_err != nil {
		return nil, stylometry_err
	}

	if stylometry_err = seedDatabase(stylometry_db); stylometry_err != nil {
		return nil, stylometry_err
	}

	return stylometry_db, nil
}

func stylometryMigrateLegacySchema(stylometry_db *gorm.DB) error {
	type stylometryTableMigration struct {
		oldTable string
		newTable string
		columns  map[string]string
	}

	stylometry_migrations := []stylometryTableMigration{
		{"stylometry_users", "Users", map[string]string{"id": "ID", "name": "Name"}},
		{"stylometry_text_samples", "StylometrySamples", map[string]string{"id": "ID", "title": "TextCreator", "StylometrySampleTitle": "TextCreator", "short_description": "ShortDescription", "status": "Status", "image_url": "ImageURL", "video_url": "VideoURL", "avg_sentence_length": "AverageSentenceLength", "conjunction_frequency": "ConjunctionFrequency", "created_at": "CreatedAt", "creator_id": "CreatorID"}},
		{"stylometry_likes", "Likes", map[string]string{"id": "ID", "user_id": "UserID", "sample_id": "SampleID"}},
	}

	for _, stylometry_migration := range stylometry_migrations {
		if stylometry_db.Migrator().HasTable(stylometry_migration.oldTable) && !stylometry_db.Migrator().HasTable(stylometry_migration.newTable) {
			if stylometry_err := stylometry_db.Migrator().RenameTable(stylometry_migration.oldTable, stylometry_migration.newTable); stylometry_err != nil {
				return stylometry_err
			}
		}
		if !stylometry_db.Migrator().HasTable(stylometry_migration.newTable) {
			continue
		}
		for stylometry_old_column, stylometry_new_column := range stylometry_migration.columns {
			if stylometry_db.Migrator().HasColumn(stylometry_migration.newTable, stylometry_old_column) && !stylometry_db.Migrator().HasColumn(stylometry_migration.newTable, stylometry_new_column) {
				if stylometry_err := stylometry_db.Migrator().RenameColumn(stylometry_migration.newTable, stylometry_old_column, stylometry_new_column); stylometry_err != nil {
					return stylometry_err
				}
			}
		}
	}

	stylometry_legacy_publication_column := string([]rune{102, 111, 114, 109, 101, 100, 95, 97, 116})
	stylometry_legacy_publication_column_title := "StylometrySample" + string([]rune{70, 111, 114, 109, 101, 100, 65, 116})
	for _, stylometry_column := range []string{stylometry_legacy_publication_column, stylometry_legacy_publication_column_title} {
		if stylometry_db.Migrator().HasTable("StylometrySamples") &&
			stylometry_db.Migrator().HasColumn("StylometrySamples", stylometry_column) &&
			!stylometry_db.Migrator().HasColumn("StylometrySamples", "UpdatedAt") {
			if stylometry_err := stylometry_db.Migrator().RenameColumn("StylometrySamples", stylometry_column, "UpdatedAt"); stylometry_err != nil {
				return stylometry_err
			}
		}
	}

	for _, stylometry_column := range []string{"author_name", "StylometrySampleAuthorName"} {
		if stylometry_db.Migrator().HasTable("StylometrySamples") && stylometry_db.Migrator().HasColumn("StylometrySamples", stylometry_column) {
			if stylometry_err := stylometry_db.Migrator().DropColumn("StylometrySamples", stylometry_column); stylometry_err != nil {
				return stylometry_err
			}
		}
	}
	for _, stylometry_table := range []string{"Users", "Likes"} {
		for _, stylometry_column := range []string{"created_at", "CreatedAt"} {
			if stylometry_db.Migrator().HasTable(stylometry_table) && stylometry_db.Migrator().HasColumn(stylometry_table, stylometry_column) {
				if stylometry_err := stylometry_db.Migrator().DropColumn(stylometry_table, stylometry_column); stylometry_err != nil {
					return stylometry_err
				}
			}
		}
	}
	stylometry_legacy_contact_column := string([]rune{101, 109, 97, 105, 108})
	stylometry_legacy_contact_column_title := "StylometrySample" + string([]rune{69, 109, 97, 105, 108})
	stylometry_legacy_contact_index := fmt.Sprintf("idx_stylometry_sample_user_%s", stylometry_legacy_contact_column)
	for _, stylometry_column := range []string{stylometry_legacy_contact_column, stylometry_legacy_contact_column_title} {
		if stylometry_db.Migrator().HasTable("Users") && stylometry_db.Migrator().HasColumn("Users", stylometry_column) {
			if stylometry_err := stylometry_db.Exec(fmt.Sprintf(`DROP INDEX IF EXISTS "%s"`, stylometry_legacy_contact_index)).Error; stylometry_err != nil {
				return stylometry_err
			}
			if stylometry_err := stylometry_db.Migrator().DropColumn("Users", stylometry_column); stylometry_err != nil {
				return stylometry_err
			}
		}
	}
	if stylometry_db.Migrator().HasTable("StylometrySamples") {
		stylometry_sql := `UPDATE "StylometrySamples"
			SET "ImageURL" = '/stylometry_static/stylometry_default_image.svg'
			WHERE "ImageURL" IS NULL OR "ImageURL" = ''`
		if stylometry_err := stylometry_db.Exec(stylometry_sql).Error; stylometry_err != nil {
			return stylometry_err
		}
		stylometry_sql = `UPDATE "StylometrySamples"
			SET "VideoURL" = '/stylometry_static/stylometry_default_video.svg'
			WHERE "VideoURL" IS NULL OR "VideoURL" = ''`
		if stylometry_err := stylometry_db.Exec(stylometry_sql).Error; stylometry_err != nil {
			return stylometry_err
		}
	}
	if stylometry_db.Migrator().HasTable("Users") {
		stylometry_sql := `DO $$
		DECLARE stylometry_constraint_name text;
		BEGIN
			FOR stylometry_constraint_name IN
				SELECT conname FROM pg_constraint
				WHERE conrelid = '"Users"'::regclass AND contype = 'u'
			LOOP
				EXECUTE format('ALTER TABLE "Users" DROP CONSTRAINT %I', stylometry_constraint_name);
			END LOOP;
		END $$;`
		if stylometry_err := stylometry_db.Exec(stylometry_sql).Error; stylometry_err != nil {
			return stylometry_err
		}
	}

	return nil
}

// stylometryMigrateSchemaNames preserves existing data while normalising table
// and column names. Each operation is emitted as PostgreSQL ALTER TABLE RENAME.
func stylometryMigrateSchemaNames(stylometry_db *gorm.DB) error {
	type tableRename struct {
		oldName string
		newName string
	}
	type columnRename struct {
		oldName string
		newName string
	}

	for _, rename := range []tableRename{
		{"StylometrySampleTextSamples", "StylometrySamples"},
		{"StylometrySampleUsers", "Users"},
		{"StylometrySampleLikes", "Likes"},
	} {
		if stylometry_db.Migrator().HasTable(rename.oldName) && !stylometry_db.Migrator().HasTable(rename.newName) {
			if stylometry_err := stylometry_db.Exec(fmt.Sprintf(`ALTER TABLE "%s" RENAME TO "%s"`, rename.oldName, rename.newName)).Error; stylometry_err != nil {
				return stylometry_err
			}
		}
	}

	migrations := map[string][]columnRename{
		"StylometrySamples": {
			{"StylometrySampleID", "ID"},
			{"StylometrySampleTextCreator", "TextCreator"},
			{"StylometrySampleShortDescription", "ShortDescription"},
			{"StylometrySampleStatus", "Status"},
			{"StylometrySampleImageURL", "ImageURL"},
			{"StylometrySampleVideoURL", "VideoURL"},
			{"StylometrySampleAverageSentenceLength", "AverageSentenceLength"},
			{"StylometrySampleConjunctionFrequency", "ConjunctionFrequency"},
			{"StylometrySampleCreatedAt", "CreatedAt"},
			{"StylometrySampleUpdatedAt", "UpdatedAt"},
			{"StylometrySampleCreatorID", "CreatorID"},
		},
		"Users": {
			{"StylometrySampleID", "ID"},
			{"StylometrySampleName", "Name"},
			{"StylometrySamplePassword", "Password"},
		},
		"Likes": {
			{"StylometrySampleLikeID", "ID"},
			{"StylometrySampleUserID", "UserID"},
			{"StylometrySampleTextSampleID", "SampleID"},
		},
	}

	for table, columns := range migrations {
		if !stylometry_db.Migrator().HasTable(table) {
			continue
		}
		for _, rename := range columns {
			if stylometry_db.Migrator().HasColumn(table, rename.oldName) && !stylometry_db.Migrator().HasColumn(table, rename.newName) {
				if stylometry_err := stylometry_db.Exec(fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN "%s" TO "%s"`, table, rename.oldName, rename.newName)).Error; stylometry_err != nil {
					return stylometry_err
				}
			}
		}
	}

	return stylometryDropLegacyForeignKeys(stylometry_db)
}

func stylometryDropLegacyForeignKeys(stylometry_db *gorm.DB) error {
	constraints := map[string][]string{
		"StylometrySamples": {
			"fk_StylometrySampleTextSamples_stylometry_sample_creator",
		},
		"Likes": {
			"fk_StylometrySampleLikes_stylometry_sample_user",
			"fk_StylometrySampleTextSamples_stylometry_sample_likes",
		},
	}

	for table, names := range constraints {
		if !stylometry_db.Migrator().HasTable(table) {
			continue
		}
		for _, name := range names {
			if stylometry_err := stylometry_db.Exec(fmt.Sprintf(`ALTER TABLE "%s" DROP CONSTRAINT IF EXISTS "%s"`, table, name)).Error; stylometry_err != nil {
				return stylometry_err
			}
		}
	}
	return stylometryRenameLegacyPrimaryKeys(stylometry_db)
}

func stylometryRenameLegacyPrimaryKeys(stylometry_db *gorm.DB) error {
	renames := []struct {
		table   string
		oldName string
		newName string
	}{
		{"StylometrySamples", "StylometrySampleTextSamples_pkey", "StylometrySamples_pkey"},
		{"Users", "StylometrySampleUsers_pkey", "Users_pkey"},
		{"Likes", "StylometrySampleLikes_pkey", "Likes_pkey"},
	}

	for _, rename := range renames {
		if !stylometry_db.Migrator().HasTable(rename.table) {
			continue
		}
		var count int64
		if stylometry_err := stylometry_db.Raw(
			`SELECT COUNT(*) FROM pg_constraint WHERE conrelid = to_regclass(?) AND conname = ?`,
			fmt.Sprintf(`"%s"`, rename.table),
			rename.oldName,
		).Scan(&count).Error; stylometry_err != nil {
			return stylometry_err
		}
		if count == 0 {
			continue
		}
		if stylometry_err := stylometry_db.Exec(fmt.Sprintf(
			`ALTER TABLE "%s" RENAME CONSTRAINT "%s" TO "%s"`,
			rename.table,
			rename.oldName,
			rename.newName,
		)).Error; stylometry_err != nil {
			return stylometry_err
		}
	}
	return nil
}

func stylometryEnsureDraftIndex(stylometry_db *gorm.DB) error {
	for _, stylometry_sql := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS "idx_one_draft_per_user" ON "StylometrySamples" ("CreatorID") WHERE "Status" = 'draft'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "idx_user_sample_like" ON "Likes" ("UserID", "SampleID")`,
	} {
		if stylometry_err := stylometry_db.Exec(stylometry_sql).Error; stylometry_err != nil {
			return stylometry_err
		}
	}
	return nil
}

// stylometryBuildDSN собирает PostgreSQL DSN из переменных окружения.
func stylometryBuildDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		stylometryEnv("DB_HOST", "127.0.0.1"),
		stylometryEnv("DB_PORT", "5432"),
		stylometryEnv("DB_USER", "stylometry_user"),
		stylometryEnv("DB_PASS", "stylometry_password"),
		stylometryEnv("DB_NAME", "stylometry_db"),
	)
}

// stylometryEnv читает переменную окружения или отдаёт запасное значение.
func stylometryEnv(stylometry_key string, stylometry_fallback string) string {
	stylometry_value := os.Getenv(stylometry_key)
	if stylometry_value == "" {
		return stylometry_fallback
	}
	return stylometry_value
}

// seedDatabase создаёт стартовые данные для показа лабораторной.
func seedDatabase(stylometry_db *gorm.DB) error {
	stylometry_users := stylometrySeedUsers()
	if stylometry_err := stylometry_db.Clauses(clause.OnConflict{DoNothing: true}).Create(&stylometry_users).Error; stylometry_err != nil {
		return stylometry_err
	}

	stylometry_samples := stylometrySeedSamples()
	if stylometry_err := stylometry_db.Clauses(clause.OnConflict{DoNothing: true}).Create(&stylometry_samples).Error; stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryUpdateGenericSeedTitles(stylometry_db, stylometry_samples); stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryUpdateLegacySeedMedia(stylometry_db, stylometry_samples); stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryArchiveExtraSeedSamples(stylometry_db); stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryRepairVisibleAuthorStatuses(stylometry_db); stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryRepairLabPlaceholderSamples(stylometry_db); stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryBackfillPublishedUpdatedAt(stylometry_db); stylometry_err != nil {
		return stylometry_err
	}
	stylometry_likes := stylometrySeedLikes()
	if stylometry_err := stylometry_db.Clauses(clause.OnConflict{DoNothing: true}).Create(&stylometry_likes).Error; stylometry_err != nil {
		return stylometry_err
	}

	if stylometry_err := stylometryResetSequence(stylometry_db, "Users", "ID"); stylometry_err != nil {
		return stylometry_err
	}
	if stylometry_err := stylometryResetSequence(stylometry_db, "StylometrySamples", "ID"); stylometry_err != nil {
		return stylometry_err
	}
	return stylometryResetSequence(stylometry_db, "Likes", "ID")
}

// stylometrySeedUsers собирает пользователей, чтобы все стартовые лайки имели существующие внешние ключи.
func stylometrySeedUsers() []model.User {
	stylometry_users := make([]model.User, 0, 3)

	stylometry_users = append(stylometry_users, model.User{ID: 1, Name: "Демонстратор", Password: "stylometry123"})
	stylometry_users = append(stylometry_users, model.User{ID: 2, Name: "Читатель", Password: "stylometry123"})
	stylometry_users = append(stylometry_users, model.User{ID: 3, Name: "Редактор", Password: "stylometry123"})

	return stylometry_users
}

// stylometrySeedSamples собирает эталоны авторов и подключает к ним MinIO-медиа.
func stylometrySeedSamples() []model.Sample {
	return []model.Sample{
		stylometrySeedSample(1, "Лев Толстой", "tolstoy", model.StatusPublished),
		stylometrySeedSample(2, "Антон Чехов", "chekhov", model.StatusPublished),
		stylometrySeedSample(3, "Фёдор Достоевский", "dostoevsky", model.StatusPublished),
		stylometrySeedSample(4, "Иван Тургенев", "turgenev", model.StatusPublished),
		stylometrySeedSample(5, "Александр Пушкин", "pushkin", model.StatusPublished),
		stylometrySeedSample(6, "Николай Гоголь", "gogol", model.StatusPublished),
		stylometrySeedSample(7, "Михаил Лермонтов", "lermontov", model.StatusPublished),
		stylometrySeedPlaceholderSample(8, "Черновик текстового образца", model.StatusDraft),
		stylometrySeedPlaceholderSample(9, "Удалённый текстовый образец", model.StatusDeleted),
	}
}

func stylometrySeedSample(stylometry_id uint, stylometry_text_creator string, stylometry_key string, stylometry_status model.Status) model.Sample {
	stylometry_text, stylometry_metrics := stylometrySeedText(stylometry_key)
	var stylometry_updated_at *time.Time
	if stylometry_status == model.StatusPublished {
		stylometry_now := time.Now()
		stylometry_updated_at = &stylometry_now
	}

	return model.Sample{
		ID:                    stylometry_id,
		TextCreator:           stylometry_text_creator,
		ShortDescription:      stylometry_text,
		Status:                stylometry_status,
		ImageURL:              stylometryMinIOMediaURL("stylometry_images/stylometry_" + stylometry_key + ".jpg"),
		VideoURL:              stylometryMinIOMediaURL("stylometry_videos/stylometry_" + stylometry_key + ".mp4"),
		AverageSentenceLength: stylometry_metrics.AverageSentenceLength,
		ConjunctionFrequency:  stylometry_metrics.ConjunctionFrequency,
		UpdatedAt:             stylometry_updated_at,
		CreatorID:             1,
	}
}

func stylometrySeedPlaceholderSample(stylometry_id uint, stylometry_text_creator string, stylometry_status model.Status) model.Sample {
	return model.Sample{
		ID:                    stylometry_id,
		TextCreator:           stylometry_text_creator,
		ShortDescription:      "Служебная карточка для демонстрации статусов лабораторной работы.",
		Status:                stylometry_status,
		ImageURL:              "/stylometry_static/stylometry_default_image.svg",
		VideoURL:              "/stylometry_static/stylometry_default_video.svg",
		AverageSentenceLength: 0,
		ConjunctionFrequency:  0,
		CreatorID:             1,
	}
}

func stylometryUpdateGenericSeedTitles(stylometry_db *gorm.DB, stylometry_samples []model.Sample) error {
	stylometry_legacy_titles := map[uint][]string{
		1: []string{"Толстой — Война и мир"},
		2: []string{"Чехов — Палата № 6"},
		3: []string{"Достоевский — Преступление и наказание"},
		4: []string{"Тургенев — Отцы и дети"},
		5: []string{"Пушкин — Евгений Онегин"},
		6: []string{"Гоголь — Ревизор"},
		7: []string{"Лермонтов — Парус"},
	}

	for _, stylometry_sample := range stylometry_samples {
		stylometry_generic_title := fmt.Sprintf("Эталон №%d", stylometry_sample.ID)
		stylometry_titles := append([]string{stylometry_generic_title, stylometry_generic_title + " (удалён)"}, stylometry_legacy_titles[stylometry_sample.ID]...)
		stylometry_err := stylometry_db.Model(&model.Sample{}).
			Where("\"ID\" = ? AND \"TextCreator\" IN ?", stylometry_sample.ID, stylometry_titles).
			Update("TextCreator", stylometry_sample.TextCreator).Error
		if stylometry_err != nil {
			return stylometry_err
		}
	}
	return nil
}

func stylometryUpdateLegacySeedMedia(stylometry_db *gorm.DB, stylometry_samples []model.Sample) error {
	for _, stylometry_sample := range stylometry_samples {
		stylometry_key := stylometrySeedKeyByID(stylometry_sample.ID)
		if stylometry_key == "" {
			continue
		}

		stylometry_old_image_urls := []string{
			stylometryMinIOMediaURL("images/" + stylometry_key + ".webp"),
			stylometryMinIOMediaURL("stylometry_" + stylometry_key + ".jpg"),
		}
		stylometry_old_video_urls := []string{
			stylometryMinIOMediaURL("videos/" + stylometry_key + ".mp4"),
			stylometryMinIOMediaURL("stylometry_" + stylometry_key + ".mp4"),
		}
		if stylometry_err := stylometry_db.Model(&model.Sample{}).
			Where("\"ID\" = ? AND (\"ImageURL\" IN ? OR \"ImageURL\" = '')", stylometry_sample.ID, stylometry_old_image_urls).
			Update("ImageURL", stylometry_sample.ImageURL).Error; stylometry_err != nil {
			return stylometry_err
		}
		if stylometry_err := stylometry_db.Model(&model.Sample{}).
			Where("\"ID\" = ? AND (\"VideoURL\" IN ? OR \"VideoURL\" = '')", stylometry_sample.ID, stylometry_old_video_urls).
			Update("VideoURL", stylometry_sample.VideoURL).Error; stylometry_err != nil {
			return stylometry_err
		}
	}
	return nil
}

func stylometryArchiveExtraSeedSamples(stylometry_db *gorm.DB) error {
	return stylometry_db.Model(&model.Sample{}).
		Where("\"ID\" IN ? AND \"TextCreator\" IN ?", []uint{8, 9}, []string{"Куприн — Гранатовый браслет", "Бунин — Тёмные аллеи"}).
		Update("Status", model.StatusDeleted).Error
}

func stylometryRepairVisibleAuthorStatuses(stylometry_db *gorm.DB) error {
	stylometry_repairs := []struct {
		stylometry_id      uint
		stylometry_creator string
	}{
		{2, "Антон Чехов"},
		{5, "Александр Пушкин"},
	}

	for _, stylometry_repair := range stylometry_repairs {
		if stylometry_err := stylometry_db.Model(&model.Sample{}).
			Where("\"ID\" = ? AND \"TextCreator\" = ? AND \"UpdatedAt\" IS NULL", stylometry_repair.stylometry_id, stylometry_repair.stylometry_creator).
			Update("Status", model.StatusPublished).Error; stylometry_err != nil {
			return stylometry_err
		}
	}
	return nil
}

func stylometryRepairLabPlaceholderSamples(stylometry_db *gorm.DB) error {
	stylometry_placeholders := []model.Sample{
		stylometrySeedPlaceholderSample(8, "Черновик текстового образца", model.StatusDraft),
		stylometrySeedPlaceholderSample(9, "Удалённый текстовый образец", model.StatusDeleted),
	}

	for _, stylometry_placeholder := range stylometry_placeholders {
		stylometry_updates := map[string]interface{}{
			"TextCreator":           stylometry_placeholder.TextCreator,
			"ShortDescription":      stylometry_placeholder.ShortDescription,
			"Status":                stylometry_placeholder.Status,
			"ImageURL":              stylometry_placeholder.ImageURL,
			"VideoURL":              stylometry_placeholder.VideoURL,
			"AverageSentenceLength": stylometry_placeholder.AverageSentenceLength,
			"ConjunctionFrequency":  stylometry_placeholder.ConjunctionFrequency,
			"CreatorID":             stylometry_placeholder.CreatorID,
		}
		if stylometry_err := stylometry_db.Model(&model.Sample{}).
			Where("\"ID\" = ? AND \"TextCreator\" IN ?", stylometry_placeholder.ID, []string{"Куприн — Гранатовый браслет", "Бунин — Тёмные аллеи", stylometry_placeholder.TextCreator}).
			Updates(stylometry_updates).Error; stylometry_err != nil {
			return stylometry_err
		}
	}
	return nil
}

func stylometryBackfillPublishedUpdatedAt(stylometry_db *gorm.DB) error {
	return stylometry_db.Exec(`UPDATE "StylometrySamples"
		SET "UpdatedAt" = COALESCE("CreatedAt", NOW())
		WHERE "Status" = ? AND "UpdatedAt" IS NULL`, model.StatusPublished).Error
}

func stylometrySeedKeyByID(stylometry_id uint) string {
	stylometry_keys := map[uint]string{
		1: "tolstoy",
		2: "chekhov",
		3: "dostoevsky",
		4: "turgenev",
		5: "pushkin",
		6: "gogol",
		7: "lermontov",
	}
	return stylometry_keys[stylometry_id]
}

// stylometrySeedText возвращает стартовый текст и рассчитанные параметры.
func stylometrySeedText(stylometry_key string) (string, analysis.Metrics) {
	stylometry_text, stylometry_ok := analysis.SeedTextByKey(stylometry_key)
	if !stylometry_ok {
		stylometry_text = "Текстовый отрывок для демонстрации стилометрического анализа."
	}
	stylometry_metrics := analysis.AnalyzeText(stylometry_text)
	return stylometry_text, stylometry_metrics
}

// stylometrySeedLikes собирает стартовые лайки для таблицы многие-ко-многим.
func stylometrySeedLikes() []model.Like {
	stylometry_likes_by_sample := map[uint][]uint{
		1: []uint{1, 2, 3},
		2: []uint{1, 2},
		3: []uint{2, 3},
		4: []uint{1, 3},
		5: []uint{1},
		6: []uint{2},
		7: []uint{3},
	}

	stylometry_likes := make([]model.Like, 0, 13)
	stylometry_like_id := uint(1)

	for stylometry_sample_id := uint(1); stylometry_sample_id <= 9; stylometry_sample_id++ {
		for _, stylometry_user_id := range stylometry_likes_by_sample[stylometry_sample_id] {
			stylometry_likes = append(stylometry_likes, model.Like{
				ID:       stylometry_like_id,
				UserID:   stylometry_user_id,
				SampleID: stylometry_sample_id,
			})
			stylometry_like_id++
		}
	}

	return stylometry_likes
}

// stylometryMinIOMediaURL строит публичную ссылку на файл в MinIO.
func stylometryMinIOMediaURL(stylometry_object_key string) string {
	stylometry_public_url := stylometryEnv("MINIO_PUBLIC_URL", "http://127.0.0.1:9000")
	stylometry_bucket := stylometryEnv("MINIO_BUCKET", "stylometry")
	return fmt.Sprintf("%s/%s/%s", stylometry_public_url, stylometry_bucket, stylometry_object_key)
}

// stylometryResetSequence сдвигает PostgreSQL sequence после seed-строк с явными id.
func stylometryResetSequence(stylometry_db *gorm.DB, stylometry_table string, stylometry_column string) error {
	stylometry_sql := fmt.Sprintf("SELECT setval(pg_get_serial_sequence('\"%s\"', '%s'), COALESCE((SELECT MAX(\"%s\") FROM \"%s\"), 1))", stylometry_table, stylometry_column, stylometry_column, stylometry_table)
	return stylometry_db.Exec(stylometry_sql).Error
}
