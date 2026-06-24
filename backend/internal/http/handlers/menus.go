package handlers

import (
	"context"
	"strconv"
	"strings"

	"dw0rdwk/backend/internal/models"
	"dw0rdwk/backend/internal/repository"

	"github.com/gofiber/fiber/v2"
)

type menuPayload struct {
	ParentID   *uint   `json:"parentId"`
	Name       *string `json:"name"`
	Route      *string `json:"route"`
	Icon       *string `json:"icon"`
	Type       *string `json:"type"`
	SortOrder  *int    `json:"sortOrder"`
	Visible    *bool   `json:"visible"`
	Permission *string `json:"permission"`
}

type menuNode struct {
	models.AdminMenu
	Children []menuNode `json:"children,omitempty"`
}

type menuSortPayload struct {
	Items []menuSortPayloadItem `json:"items"`
}

type menuSortPayloadItem struct {
	ID        uint `json:"id"`
	ParentID  uint `json:"parentId"`
	SortOrder int  `json:"sortOrder"`
}

func Menus(repo *repository.MenuRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		items, err := repo.List(c.UserContext(), c.QueryBool("all") || c.QueryBool("includeHidden"))
		if err != nil {
			return err
		}
		return OK(c, buildMenuTree(items))
	}
}

func CreateMenu(repo *repository.MenuRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req menuPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		item := menuFromPayload(req)
		if err := validateMenuCandidate(c.UserContext(), repo, 0, item); err != nil {
			return err
		}
		if err := repo.Create(c.UserContext(), &item); err != nil {
			return err
		}
		auditLog(c, logs, "admin_menu_create", auditText("menu", item.ID, "name="+item.Name+" route="+item.Route), 0)
		return OK(c, item)
	}
}

func UpdateMenu(repo *repository.MenuRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		current, err := repo.Find(c.UserContext(), id)
		if err != nil {
			return err
		}
		var req menuPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		next, values := applyMenuPayload(current, req)
		if err := validateMenuCandidate(c.UserContext(), repo, id, next); err != nil {
			return err
		}
		if err := repo.Update(c.UserContext(), id, values); err != nil {
			return err
		}
		auditLog(c, logs, "admin_menu_update", auditText("menu", id, auditFields(values)), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func DeleteMenu(repo *repository.MenuRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := pathID(c)
		if err != nil {
			return err
		}
		hasChildren, err := repo.HasChildren(c.UserContext(), id)
		if err != nil {
			return err
		}
		if hasChildren {
			return fiber.NewError(fiber.StatusBadRequest, "menu has children")
		}
		if err := repo.Delete(c.UserContext(), id); err != nil {
			return err
		}
		auditLog(c, logs, "admin_menu_delete", auditText("menu", id, ""), 0)
		return OK(c, fiber.Map{"id": id})
	}
}

func SortMenus(repo *repository.MenuRepository, logs *repository.LogRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req menuSortPayload
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		if len(req.Items) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "items are required")
		}
		if err := validateMenuSortPayload(c.UserContext(), repo, req.Items); err != nil {
			return err
		}
		items := make([]repository.MenuSortItem, 0, len(req.Items))
		for _, item := range req.Items {
			items = append(items, repository.MenuSortItem{
				ID:        item.ID,
				ParentID:  item.ParentID,
				SortOrder: item.SortOrder,
			})
		}
		if err := repo.UpdateSort(c.UserContext(), items); err != nil {
			return err
		}
		auditLog(c, logs, "admin_menu_sort", "menus count="+strconv.Itoa(len(items)), 0)
		return OK(c, fiber.Map{"ok": true})
	}
}

func buildMenuTree(items []models.AdminMenu) []menuNode {
	children := map[uint][]models.AdminMenu{}
	for _, item := range items {
		children[item.ParentID] = append(children[item.ParentID], item)
	}
	var build func(parentID uint) []menuNode
	build = func(parentID uint) []menuNode {
		nodes := make([]menuNode, 0, len(children[parentID]))
		for _, item := range children[parentID] {
			node := menuNode{AdminMenu: item}
			node.Children = build(item.ID)
			nodes = append(nodes, node)
		}
		return nodes
	}
	return build(0)
}

func menuFromPayload(req menuPayload) models.AdminMenu {
	item := models.AdminMenu{
		ParentID:   uintValue(req.ParentID),
		Name:       trimStringPtr(req.Name),
		Route:      trimStringPtr(req.Route),
		Icon:       trimStringPtr(req.Icon),
		Type:       normalizeMenuType(trimStringPtr(req.Type)),
		SortOrder:  intValue(req.SortOrder, 10),
		Visible:    boolValue(req.Visible, true),
		Permission: normalizeMenuPermission(trimStringPtr(req.Permission)),
	}
	return item
}

func applyMenuPayload(current models.AdminMenu, req menuPayload) (models.AdminMenu, map[string]any) {
	values := map[string]any{}
	next := current
	if req.ParentID != nil {
		next.ParentID = *req.ParentID
		values["parent_id"] = next.ParentID
	}
	if req.Name != nil {
		next.Name = trimStringPtr(req.Name)
		values["name"] = next.Name
	}
	if req.Route != nil {
		next.Route = trimStringPtr(req.Route)
		values["route"] = next.Route
	}
	if req.Icon != nil {
		next.Icon = trimStringPtr(req.Icon)
		values["icon"] = next.Icon
	}
	if req.Type != nil {
		next.Type = normalizeMenuType(trimStringPtr(req.Type))
		values["type"] = next.Type
	}
	if req.SortOrder != nil {
		next.SortOrder = *req.SortOrder
		values["sort_order"] = next.SortOrder
	}
	if req.Visible != nil {
		next.Visible = *req.Visible
		values["visible"] = next.Visible
	}
	if req.Permission != nil {
		next.Permission = normalizeMenuPermission(trimStringPtr(req.Permission))
		values["permission"] = next.Permission
	}
	return next, values
}

func validateMenuCandidate(ctx context.Context, repo *repository.MenuRepository, id uint, item models.AdminMenu) error {
	if strings.TrimSpace(item.Name) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if item.Type != "menu" && item.Type != "dir" {
		return fiber.NewError(fiber.StatusBadRequest, "type must be menu or dir")
	}
	if item.ParentID != 0 {
		exists, err := repo.ParentExists(ctx, item.ParentID)
		if err != nil {
			return err
		}
		if !exists {
			return fiber.NewError(fiber.StatusBadRequest, "parent menu not found")
		}
	}
	cycle, err := repo.WouldCreateCycle(ctx, id, item.ParentID)
	if err != nil {
		if repository.IsNotFound(err) {
			return fiber.NewError(fiber.StatusBadRequest, "parent menu not found")
		}
		return err
	}
	if cycle {
		return fiber.NewError(fiber.StatusBadRequest, "parent menu creates a cycle")
	}
	if len([]rune(item.Name)) > 80 {
		return fiber.NewError(fiber.StatusBadRequest, "name must be 80 characters or fewer")
	}
	if len(item.Route) > 160 {
		return fiber.NewError(fiber.StatusBadRequest, "route must be 160 characters or fewer")
	}
	if len(item.Icon) > 80 {
		return fiber.NewError(fiber.StatusBadRequest, "icon must be 80 characters or fewer")
	}
	if len(item.Permission) > 80 {
		return fiber.NewError(fiber.StatusBadRequest, "permission must be 80 characters or fewer")
	}
	return nil
}

func validateMenuSortPayload(ctx context.Context, repo *repository.MenuRepository, items []menuSortPayloadItem) error {
	menus, err := repo.List(ctx, true)
	if err != nil {
		return err
	}
	known := map[uint]struct{}{}
	parentMap := map[uint]uint{}
	for _, item := range menus {
		known[item.ID] = struct{}{}
		parentMap[item.ID] = item.ParentID
	}
	seenInput := map[uint]struct{}{}
	for _, item := range items {
		if item.ID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "item id is required")
		}
		if _, ok := known[item.ID]; !ok {
			return fiber.NewError(fiber.StatusBadRequest, "menu not found")
		}
		if _, ok := seenInput[item.ID]; ok {
			return fiber.NewError(fiber.StatusBadRequest, "duplicate menu id")
		}
		seenInput[item.ID] = struct{}{}
		if item.ParentID != 0 {
			if _, ok := known[item.ParentID]; !ok {
				return fiber.NewError(fiber.StatusBadRequest, "parent menu not found")
			}
		}
		parentMap[item.ID] = item.ParentID
	}
	for id := range parentMap {
		visited := map[uint]struct{}{}
		current := parentMap[id]
		for current != 0 {
			if current == id {
				return fiber.NewError(fiber.StatusBadRequest, "parent menu creates a cycle")
			}
			if _, ok := visited[current]; ok {
				return fiber.NewError(fiber.StatusBadRequest, "parent menu creates a cycle")
			}
			visited[current] = struct{}{}
			current = parentMap[current]
		}
	}
	return nil
}

func normalizeMenuType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "menu"
	}
	if value == "directory" || value == "group" {
		return "dir"
	}
	return value
}

func normalizeMenuPermission(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "admin"
	}
	return value
}

func trimStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func uintValue(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
