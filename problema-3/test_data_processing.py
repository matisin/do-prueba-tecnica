import pandas as pd
import pytest
from data_processing import (load_data, join_data, standardize_dates,
                       standardize_names, consolidate_data, standarize_variable,
                       crucero_con_mas_data, hora_con_mas_muestreos,
                       promedios_por_profundidad)

def test_load_data():
    viajes, mediciones = load_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
    assert isinstance(viajes, pd.DataFrame)
    assert isinstance(mediciones, pd.DataFrame)
    assert 'id' in viajes.columns
    assert 'id' in mediciones.columns

def test_join_data():
    viajes, mediciones = load_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
    data = join_data(viajes, mediciones)
    assert len(data) == 119608
    assert 'Cruise' in data.columns
    assert 'valor' in data.columns

def test_standarize_dates():
    viajes, mediciones = load_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
    data = join_data(viajes, mediciones)

    data = standardize_dates(data)
    assert data['standarized_date'].nunique() == 52

def test_standarize_nombres():
    viajes, mediciones = load_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
    data = join_data(viajes, mediciones)
    data = standardize_dates(data)
    data = standardize_names(data)

    assert 'longitude__degrees_east_' in data.columns
    assert 'bot__depth__m_' in data.columns
    assert 'depth__m_' in data.columns

def test_standarize_variables():
    viajes, mediciones = load_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
    data = join_data(viajes, mediciones)
    data = standardize_dates(data)
    data = standardize_names(data)
    data = standarize_variable(data)

    assert 'variable' in data.columns
    assert data['variable'].iloc[0] == 'fluorescence__wet_labs_eco-afl_fl__mg_m_3_'

def test_crucero_con_mas_data():
    data = consolidate_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
    crucero = crucero_con_mas_data(data)

    assert crucero == 'Crucero 2017'
    print("el crucero con mas data es {0}".format(crucero))

def test_hora_con_mas_muestreos():
    data = consolidate_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")

    assert hora_con_mas_muestreos(data) == 20

def test_promedios_por_profundidad():
    data = consolidate_data("data/viajes_dd8a0ac9e2.csv", "data/mediciones_4be6910e87.csv")
    promedios = promedios_por_profundidad(data)
    assert not promedios.empty
