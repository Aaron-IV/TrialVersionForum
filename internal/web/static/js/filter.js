document.getElementById('apply-filter').addEventListener('click', () => {
    const checked = Array.from(document.querySelectorAll('input[name="category"]:checked'))
        .map(cb => cb.value);
    
    // Имитируем результат фильтрации
    document.getElementById('filter-results').innerHTML =
        `<p>Выбранные категории: ${checked.join(', ')}</p>`;
});
