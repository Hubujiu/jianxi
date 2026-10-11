package blog

import "strings"

func validateTaxonomyMetadata(m TaxonomyMetadata) error {
	if strings.TrimSpace(m.Name) == "" || len(m.Name) > 120 || len(m.ID) > 64 || len(m.Description) > 2000 {
		return fail("INVALID_TAXONOMY", "分类/标签元数据必须提供名称，名称最长 120 字节，说明最长 2000 字节")
	}
	if !slugPattern.MatchString(m.Slug) || len(m.Slug) > 100 {
		return fail("INVALID_TAXONOMY", "分类/标签元数据必须明确提供 slug，使用小写字母、数字和连字符，最长 100 字符；不自动生成路由")
	}
	return nil
}

func taxonomyMetadata(t Taxon) TaxonomyMetadata {
	return TaxonomyMetadata{ID: t.ID, Name: t.Name, Slug: t.Slug, Description: t.Description}
}

// Legacy revisions remain unchanged (including their source hashes). Working
// copies expose current taxonomy metadata, so they can be uploaded under the new
// contract after a category/tag is renamed or its route is changed.
func contentWithTaxonomyMetadata(c Content, tax map[string]Taxon) Content {
	c.Category = nil
	c.Tags = nil
	if t, ok := tax[c.CategoryID]; ok && !t.Deleted {
		m := taxonomyMetadata(t)
		c.Category = &m
	}
	for _, id := range c.TagIDs {
		if t, ok := tax[id]; ok && !t.Deleted {
			c.Tags = append(c.Tags, taxonomyMetadata(t))
		}
	}
	return c
}

// Resolve IDs without creating candidates. Failed/abandoned uploads and parse
// failures must not create categories or tags. New candidates are committed in
// the same transaction as the parsed article draft.
func normalizeUploadTaxonomy(c Content, tax map[string]Taxon) (Content, error) {
	c = clone(c)
	resolve := func(m TaxonomyMetadata, kind, legacyID string) (TaxonomyMetadata, error) {
		if err := validateTaxonomyMetadata(m); err != nil {
			return m, err
		}
		if m.ID != "" && legacyID != "" && m.ID != legacyID {
			return m, fail("TAXONOMY_CONFLICT", "元数据 ID 与文章关联 ID 不一致")
		}
		if m.ID == "" {
			m.ID = legacyID
		}
		if m.ID != "" {
			t, ok := tax[m.ID]
			if !ok || t.Deleted || t.Kind != kind {
				return m, fail("TAXONOMY_MISSING", "元数据引用的分类/标签不存在")
			}
			if t.Name != m.Name || t.Slug != m.Slug {
				return m, fail("TAXONOMY_CONFLICT", "分类/标签名称或路由已改变；请重新读取元数据，改路由须单独提交 taxonomy 变更")
			}
			return taxonomyMetadata(t), nil
		}
		for _, t := range tax {
			if t.Kind != kind || t.Deleted {
				continue
			}
			if t.Name == m.Name || t.Slug == m.Slug {
				if t.Name != m.Name || t.Slug != m.Slug {
					return m, fail("TAXONOMY_CONFLICT", "分类/标签名称或路由已被其他元数据使用")
				}
				return taxonomyMetadata(t), nil
			}
		}
		m.ID = uid()
		return m, nil
	}
	if c.Category == nil && c.CategoryID != "" {
		return c, fail("ROUTE_METADATA_REQUIRED", "文章分类必须提供 category 的 name 和 slug，不能只传 categoryId")
	}
	if c.Category != nil {
		m, err := resolve(*c.Category, "category", c.CategoryID)
		if err != nil {
			return c, err
		}
		c.Category, c.CategoryID = &m, m.ID
	}
	if len(c.TagIDs) > 0 && len(c.TagIDs) != len(c.Tags) {
		return c, fail("ROUTE_METADATA_REQUIRED", "文章标签必须逐个提供 tags 的 name 和 slug，不能只传 tagIds")
	}
	ids, names, slugs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for i, m := range c.Tags {
		legacyID := ""
		if len(c.TagIDs) > 0 {
			legacyID = c.TagIDs[i]
		}
		m, err := resolve(m, "tag", legacyID)
		if err != nil {
			return c, err
		}
		if ids[m.ID] || names[m.Name] || slugs[m.Slug] {
			return c, fail("TAXONOMY_CONFLICT", "文章标签元数据重复")
		}
		ids[m.ID], names[m.Name], slugs[m.Slug] = true, true, true
		c.Tags[i] = m
	}
	c.TagIDs = []string{}
	for _, m := range c.Tags {
		c.TagIDs = append(c.TagIDs, m.ID)
	}
	return c, nil
}

func commitUploadTaxonomy(c Content, tax map[string]Taxon) error {
	commit := func(m TaxonomyMetadata, kind string) error {
		if t, ok := tax[m.ID]; ok {
			if t.Deleted || t.Kind != kind || t.Name != m.Name || t.Slug != m.Slug {
				return fail("TAXONOMY_CONFLICT", "分类/标签元数据已改变，请重新获取副本并上传")
			}
			return nil
		}
		for _, t := range tax {
			if !t.Deleted && t.Kind == kind && (t.Name == m.Name || t.Slug == m.Slug) {
				return fail("TAXONOMY_CONFLICT", "分类/标签已由其他草稿创建，请重新读取元数据并上传")
			}
		}
		tax[m.ID] = Taxon{ID: m.ID, Kind: kind, Name: m.Name, Slug: m.Slug, Description: m.Description, Revision: 1}
		return nil
	}
	if c.Category != nil {
		if err := commit(*c.Category, "category"); err != nil {
			return err
		}
	}
	for _, m := range c.Tags {
		if err := commit(m, "tag"); err != nil {
			return err
		}
	}
	return nil
}
