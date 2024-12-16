package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    "os"
    "strings"
    "html"
    "runtime"
)

const (
    BaseURL = "https://loverepublic.ru"
    APIEndpoint = BaseURL + "/api/catalog"
    ProductsPerPage = 12
    RequestTimeout = 30 * time.Second
    DelayBetweenRequests = 1 * time.Second
    OutputFileName = "loverepublic_products.json"
)

// Структуры для JSON
type CategoryResponse struct {
    Status int `json:"status"`
    Data struct {
        Variables struct {
            SectionId   int    `json:"sectionId"`
            SectionCode string `json:"sectionCode"` 
            SectionName string `json:"sectionName"`
        } `json:"variables"`
    } `json:"data"`
}

type ProductsResponse struct {
    Status int `json:"status"`
    Data struct {
        Items []Product `json:"items"`
        Pagination struct {
            Total  int `json:"total"`
            Pages  int `json:"pages"`
            Limit  int `json:"limit"`
        } `json:"pagination"`
    } `json:"data"`
}

type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Article     string  `json:"article"`
    Description string  `json:"description"`
    Color       Color   `json:"color"`
    Price       Price   `json:"price"`
    SKU         []SKU   `json:"sku"`
    Images      Images  `json:"images"`
}

type Color struct {
    ColorName   string `json:"colorName"`
    ColorCommon string `json:"colorCommon"`
    ColorHex    string `json:"colorHex"`
}

type Price struct {
    Value           int    `json:"value"`
    DiscountValue   int    `json:"discountValue"`
    DiscountPercent int    `json:"discountPercent"`
    Currency        string `json:"currency"`
}

type SKU struct {
    ID       int    `json:"id"`
    Size     string `json:"size"`
    Quantity int    `json:"quantity"`
    Stock    []struct {
        StoreID  int `json:"storeId"`
        Quantity int `json:"quantity"`
    } `json:"stock"`
}

type Images struct {
    Original []string `json:"original"`
    Detail   []string `json:"detail"`
}

// HTTP клиент с таймаутом
var client = &http.Client{
    Timeout: RequestTimeout,
}

// Получение данных категории
func getCategory(categoryURL string) (*CategoryResponse, error) {
    url := fmt.Sprintf("%s?url=%s", APIEndpoint, categoryURL)
    
    resp, err := client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("ошибка при запросе категории: %w", err)
    }
    defer resp.Body.Close()

    var category CategoryResponse
    if err := json.NewDecoder(resp.Body).Decode(&category); err != nil {
        return nil, fmt.Errorf("ошибка при декодировании ответа категории: %w", err)
    }

    return &category, nil
}

// Получение списка продуктов
func getProducts(sectionID int, page int) (*ProductsResponse, error) {
    body := map[string]interface{}{
        "uri":    fmt.Sprintf("/catalog/sections/%d/products/", sectionID),
        "city":   "Москва",
        "order":  "DESC",
        "sort":   "SORT",
        "offset": (page - 1) * ProductsPerPage,
        "limit":  ProductsPerPage,
    }
    
    jsonBody, err := json.Marshal(body)
    if err != nil {
        return nil, fmt.Errorf("ошибка при создании тела запроса: %w", err)
    }

    req, err := http.NewRequest("POST", APIEndpoint, bytes.NewBuffer(jsonBody))
    if err != nil {
        return nil, fmt.Errorf("ошибка при создании запроса: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
    }
    defer resp.Body.Close()

    var products ProductsResponse
    if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
        return nil, fmt.Errorf("ошибка при декодировании ответа продуктов: %w", err)
    }

    return &products, nil
}

// Очистка описания продукта
func cleanDescription(desc string) string {
    desc = html.UnescapeString(desc)
    
    // Удаляем HTML-теги и заменяем переносы строк
    replacements := map[string]string{
        "<ul>": "",
        "</ul>": "",
        "<li>": "",
        "</li>": " ",
        "\\n": " ",
        "\\r": " ",
        "\n": " ",
        "\r": " ",
    }
    
    for old, new := range replacements {
        desc = strings.ReplaceAll(desc, old, new)
    }
    
    // Нормализация пробелов
    for strings.Contains(desc, "  ") {
        desc = strings.ReplaceAll(desc, "  ", " ")
    }
    
    return strings.TrimSpace(desc)
}

// Подготовка файла для записи
func prepareOutputFile() (*os.File, error) {
    // Удаляем старый файл если существует
    if _, err := os.Stat(OutputFileName); err == nil {
        if err := os.Remove(OutputFileName); err != nil {
            return nil, fmt.Errorf("ошибка при удалении старого файла: %w", err)
        }
        fmt.Println("Старый файл удален")
    }

    file, err := os.Create(OutputFileName)
    if err != nil {
        return nil, fmt.Errorf("ошибка при создании файла: %w", err)
    }

    return file, nil
}

func main() {
    // Подготовка файла
    file, err := prepareOutputFile()
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    // Получение данных категории
    category, err := getCategory(BaseURL + "/catalog/odezhda/")
    if err != nil {
        log.Fatal(err)
    }

    // Получение первой страницы для определения общего количества
    firstPage, err := getProducts(category.Data.Variables.SectionId, 1)
    if err != nil {
        log.Fatal(err)
    }

    totalPages := firstPage.Data.Pagination.Pages
    totalProducts := 0

    // Запись начала JSON массива
    file.WriteString("[\n")

    // Обработка всех страниц
    for page := 1; page <= totalPages; page++ {
        products, err := getProducts(category.Data.Variables.SectionId, page)
        if err != nil {
            log.Printf("Ошибка при получении страницы %d: %v", page, err)
            continue
        }

        // Очистка описаний
        for i := range products.Data.Items {
            products.Data.Items[i].Description = cleanDescription(products.Data.Items[i].Description)
        }

        // Запись в файл
        jsonData, err := json.MarshalIndent(products.Data.Items, "    ", "    ")
        if err != nil {
            log.Printf("Ошибка при маршалинге JSON для страницы %d: %v", page, err)
            continue
        }

        if page > 1 {
            file.WriteString(",\n")
        }
        file.Write(jsonData[1:len(jsonData)-1])

        totalProducts += len(products.Data.Items)
        fmt.Printf("Обработана страница %d из %d (всего товаров: %d)\n", 
            page, totalPages, totalProducts)

        // Очистка памяти
        products = nil
        jsonData = nil
        runtime.GC()

        time.Sleep(DelayBetweenRequests)
    }

    // Закрытие JSON массива
    file.WriteString("\n]")

    fmt.Printf("Парсинг завершен. Сохранено %d товаров в %s\n", 
        totalProducts, OutputFileName)
}