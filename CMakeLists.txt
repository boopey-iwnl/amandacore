if(NOT PROJECT_NAME)
    cmake_minimum_required(VERSION 3.24)

    function(get_engine_finder_cmake_path project_json_file_path path_value)
        if(NOT ${path_value} AND EXISTS "${project_json_file_path}")
            file(READ "${project_json_file_path}" project_json_data)
            string(JSON engine_finder_cmake_value ERROR_VARIABLE json_error GET ${project_json_data} "engine_finder_cmake_path")
            cmake_path(APPEND CMAKE_CURRENT_SOURCE_DIR "${engine_finder_cmake_value}" engine_finder_cmake_value)
            if(NOT json_error AND EXISTS "${engine_finder_cmake_value}")
                set(${path_value} "${engine_finder_cmake_value}" PARENT_SCOPE)
            elseif(json_error AND ${engine_finder_cmake_value} STREQUAL "NOTFOUND")
                message(WARNING "Unable to read 'engine_finder_cmake_path'.\nError: ${json_error} ${engine_finder_cmake_value}")
            endif()
        endif()
    endfunction()

    get_engine_finder_cmake_path("${CMAKE_CURRENT_SOURCE_DIR}/user/project.json" engine_finder_cmake_path)
    get_engine_finder_cmake_path("${CMAKE_CURRENT_SOURCE_DIR}/project.json" engine_finder_cmake_path)
    if(NOT engine_finder_cmake_path)
        set(engine_finder_cmake_path cmake/EngineFinder.cmake)
    endif()

    include(${engine_finder_cmake_path} OPTIONAL)
    find_package(o3de REQUIRED)

    project(amandacore
        VERSION 0.2.0
        DESCRIPTION "amandacore clean-room O3DE MMO foundation"
        LANGUAGES C CXX)

    o3de_initialize()
endif()

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

option(MMO_REPLICA_ENABLE_TESTS "Build lightweight gameplay domain tests." ON)

if(MMO_REPLICA_ENABLE_TESTS)
    enable_testing()
endif()

if(NOT TARGET AmandaCoreShared)
    add_subdirectory(Shared/AmandaCoreShared)
endif()
