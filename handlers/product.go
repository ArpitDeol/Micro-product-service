package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"product-service/db"
	"product-service/middleware"
	"product-service/models"
)

// AddProduct — JWT required. Any logged-in user can create a product.
func AddProduct(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	var input models.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if input.Name == "" || input.Price <= 0 {
		writeError(w, http.StatusBadRequest, "Name and a positive price are required")
		return
	}

	var product models.Product
	err := db.Pool.QueryRow(r.Context(), `
		INSERT INTO products (name, description, price, category, stock_quantity, image_url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, description, price, category, stock_quantity, image_url, created_by, created_at
	`, input.Name, input.Description, input.Price, input.Category, input.Stock, input.ImageURL, userID,
	).Scan(&product.ID, &product.Name, &product.Description, &product.Price,
		&product.Category, &product.Stock, &product.ImageURL, &product.CreatedBy, &product.CreatedAt)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create product")
		return
	}

	writeJSON(w, http.StatusOK, product)
}

// DeleteProduct — JWT required. Soft delete (is_active = false), never a hard DELETE.
func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(middleware.UserIDKey).(int)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Missing or invalid token")
		return
	}

	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product id")
		return
	}

	tag, err := db.Pool.Exec(r.Context(), `
		UPDATE products SET is_active = false, updated_at = now() WHERE id = $1 AND is_active = true
	`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete product")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Product not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Product deleted"})
}

// ListProducts — public, no auth. Supports pagination + basic filters.
func ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(q.Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	category := q.Get("category")
	search := q.Get("search")

	rows, err := db.Pool.Query(r.Context(), `
		SELECT id, name, description, price, category, stock_quantity, image_url, created_by, created_at
		FROM products
		WHERE is_active = true
		  AND ($1 = '' OR category = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%')
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, category, search, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price,
			&p.Category, &p.Stock, &p.ImageURL, &p.CreatedBy, &p.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read products")
			return
		}
		products = append(products, p)
	}

	writeJSON(w, http.StatusOK, products)
}

// GetProduct — public, no auth. Single product by id.
func GetProduct(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product id")
		return
	}

	var p models.Product
	err = db.Pool.QueryRow(r.Context(), `
		SELECT id, name, description, price, category, stock_quantity, image_url, created_by, created_at
		FROM products WHERE id = $1 AND is_active = true
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Price,
		&p.Category, &p.Stock, &p.ImageURL, &p.CreatedBy, &p.CreatedAt)

	if err != nil {
		writeError(w, http.StatusNotFound, "Product not found")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

type StockAdjustInput struct {
	Delta int `json:"delta"` // positive = add stock back, negative = decrement
}

// AdjustStock — internal endpoint, called by order-service during checkout
// and during Saga rollback. Not exposed to the public dashboard.
func AdjustStock(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid product id")
		return
	}

	var input StockAdjustInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tag, err := db.Pool.Exec(r.Context(), `
		UPDATE products
		SET stock_quantity = stock_quantity + $1, updated_at = now()
		WHERE id = $2 AND is_active = true AND stock_quantity + $1 >= 0
	`, input.Delta, id)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to adjust stock")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusBadRequest, "Insufficient stock or product not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Stock adjusted"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}